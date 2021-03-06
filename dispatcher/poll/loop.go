package poll

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/venicegeo/pzsvc-exec/dispatcher/cfwrapper"
	"github.com/venicegeo/pzsvc-exec/dispatcher/model"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

var defaultTaskDiskMB = 6142
var defaultTaskMemoryMB = 4096

var pzsvcGetS3FileSizeInMegabytes = pzsvc.GetS3FileSizeInMegabytes
var pzsvcRequestKnownJSON = pzsvc.RequestKnownJSON
var pzsvcSendExecResultNoData = pzsvc.SendExecResultNoData

func init() {
	// Update defaults if overridden via env variables
	if diskMB, err := strconv.Atoi(os.Getenv("TASK_DISK_MB")); diskMB > 0 && err == nil {
		defaultTaskDiskMB = diskMB
	}
	if memoryMB, err := strconv.Atoi(os.Getenv("TASK_MEMORY_MB")); memoryMB > 0 && err == nil {
		defaultTaskMemoryMB = memoryMB
	}
}

// Loop is an encapsulation of configuration and functionality needed for a job polling loop
type Loop struct {
	PzSession     *pzsvc.Session
	PzConfig      pzsvc.Config
	SvcID         string
	ConfigPath    string
	ClientFactory cfwrapper.Factory
	vcapID        string
	taskLimit     int
	intervalTick  time.Duration

	stopChan         chan bool
	runIterationFunc func(l Loop) error
}

// NewLoop creates a Loop and does starting configuration based on the given parameters
func NewLoop(s *pzsvc.Session, configObj pzsvc.Config, svcID string, configPath string, clientFactory cfwrapper.Factory) (*Loop, error) {
	pzsvc.LogInfo(*s, "Initializing polling loop object")

	appID, err := getVCAPApplicationID()
	if err != nil {
		return nil, err
	}
	pzsvc.LogInfo(*s, "Found application name from VCAP Tree: "+appID)

	// Read the # of simultaneous Tasks that are allowed to be run by the Dispatcher
	taskLimit := 3
	if envTaskLimit := os.Getenv("TASK_LIMIT"); envTaskLimit != "" {
		taskLimit, _ = strconv.Atoi(envTaskLimit)
	}

	return &Loop{
		PzSession:        s,
		PzConfig:         configObj,
		SvcID:            svcID,
		ConfigPath:       configPath,
		ClientFactory:    clientFactory,
		vcapID:           appID,
		taskLimit:        taskLimit,
		intervalTick:     5 * time.Second,
		stopChan:         nil, // initialized when loop starts
		runIterationFunc: runIteration,
	}, nil
}

// Start begins the polling interval loop and returns a channel that feeds
// through any errors encountered in each interval
func (l *Loop) Start() <-chan error {
	errChan := make(chan error)
	l.stopChan = make(chan bool)
	go func() {
		ticker := time.Tick(l.intervalTick)
		for {
			select {
			case <-ticker:
				err := l.runIterationFunc(*l)
				if err != nil {
					errChan <- err
				}
			case <-l.stopChan:
				close(errChan)
				return
			}
		}
	}()
	return errChan
}

// Stop halts the loop's iteration
func (l Loop) Stop() {
	l.stopChan <- true
	close(l.stopChan)
}

func runIteration(l Loop) error {
	pzsvc.LogInfo(*l.PzSession, "Starting polling loop iteration")

	cfSession, err := l.ClientFactory.GetSession()
	if err != nil {
		pzsvc.LogSimpleErr(*l.PzSession, "Error generating valid CF Client", err)
		return err
	}

	numTasks, err := cfSession.CountTasksForApp(l.vcapID)
	if err != nil {
		pzsvc.LogSimpleErr(*l.PzSession, "Error checking running tasks. ", err)
		return err
	}
	if numTasks >= l.taskLimit {
		pzsvc.LogInfo(*l.PzSession, "Too many tasks already running, skipping this iteration cycle")
		return nil
	}

	taskItem, _, err := l.getPzTaskItem()
	if err != nil {
		return err
	}

	jobID := taskItem.Data.SvcData.JobID
	jobData := taskItem.Data.SvcData.Data.DataInputs.Body.Content
	if jobData == "" {
		pzsvc.LogInfo(*l.PzSession, ("No jobs available in task queue (jobID=''); skipping this iteration cycle"))
		return nil
	}
	pzsvc.LogInfo(*l.PzSession, "New Task Grabbed.  JobID: "+jobID)

	jobInput, err := l.parseJobInput(jobData)
	if err != nil {
		return err
	}

	workerCommand, err := l.buildWorkerCommand(jobInput, jobID)
	if err != nil {
		return err
	}

	diskMB, memoryMB := l.calculateDiskAndMemoryLimits(jobInput)

	taskRequest := cfwrapper.TaskRequest{
		Command:          workerCommand,
		Name:             jobID,
		DropletGUID:      l.vcapID,
		DiskInMegabyte:   diskMB,
		MemoryInMegabyte: memoryMB,
	}

	serializedInput, _ := json.Marshal(jobInput)
	pzsvc.LogAudit(*l.PzSession, l.PzSession.UserID, "Creating CF Task for Job "+jobID+" : "+workerCommand, l.PzSession.AppName, string(serializedInput), pzsvc.INFO)

	if err = cfSession.CreateTask(taskRequest); err != nil {
		if cfwrapper.IsMemoryLimitError(err) {
			pzsvc.LogAudit(*l.PzSession, l.PzSession.UserID, "Audit failure", l.PzSession.AppName, "The Memory limit of CF Org has been exceeded. No further jobs can be created.", pzsvc.ERROR)
			return errors.New("CF memory limit hit, will retry job later")
		}
		// General error - fail the job.
		pzsvc.LogAudit(*l.PzSession, l.PzSession.UserID, "Audit failure", l.PzSession.AppName, "Could not Create PCF Task for Job. Job Failed: "+err.Error(), pzsvc.ERROR)
		pzsvcSendExecResultNoData(*l.PzSession, l.PzSession.PzAddr, l.SvcID, jobID, pzsvc.PiazzaStatusFail)
		return err
	}

	return nil
}

func (l Loop) getPzTaskItem() (*model.PzTaskItem, []byte, error) {
	var pzTaskItem model.PzTaskItem
	url := fmt.Sprintf("%s/service/%s/task", l.PzSession.PzAddr, l.SvcID)

	byts, err := pzsvcRequestKnownJSON("POST", "", url, l.PzSession.PzAuth, &pzTaskItem)
	if err != nil {
		err.Log(*l.PzSession, "Dispatcher: error getting new task:"+string(byts))
		return nil, nil, err
	}
	return &pzTaskItem, byts, nil
}

func (l Loop) parseJobInput(jobInputStr string) (*pzsvc.InpStruct, error) {
	var err error
	var jobInputContent pzsvc.InpStruct

	if err = json.Unmarshal([]byte(jobInputStr), &jobInputContent); err != nil {
		pzsvc.LogSimpleErr(*l.PzSession, "Error decoding job input body", err)
		return nil, err
	}

	if jobInputContent.ExtAuth != "" {
		jobInputContent.ExtAuth = "*****"
	}
	if jobInputContent.PzAuth != "" {
		jobInputContent.PzAuth = "*****"
	}

	return &jobInputContent, nil
}

func (l Loop) buildWorkerCommand(jobInput *pzsvc.InpStruct, jobID string) (string, error) {
	workerCommand := fmt.Sprintf("worker --cliExtra '%s' --userID '%s' --config '%s' --serviceID '%s' --jobID '%s'",
		jobInput.Command, jobInput.UserID, l.ConfigPath, l.SvcID, jobID)

	if len(jobInput.InExtFiles) != len(jobInput.InExtNames) {
		return "", errors.New("Number of input file names and URLs did not match")
	}

	commandParts := []string{workerCommand}

	for i := range jobInput.InExtNames {
		inputPair := fmt.Sprintf("%s:%s", jobInput.InExtNames[i], jobInput.InExtFiles[i])
		commandParts = append(commandParts, "-i", inputPair)
	}

	for _, outputFile := range jobInput.OutGeoJs { // TODO: non-geojson outputs?
		commandParts = append(commandParts, "-o", outputFile)
	}

	return strings.Join(commandParts, " "), nil
}

func (l Loop) calculateAWSInputFileSizeMB(jobInput *pzsvc.InpStruct) (total int) {
	for _, url := range jobInput.InExtFiles {
		if strings.Contains(url, "amazonaws") {
			fileSize, err := pzsvcGetS3FileSizeInMegabytes(url)
			if err == nil {
				pzsvc.LogInfo(*l.PzSession, fmt.Sprintf("S3 File Size for %s found to be %d", url, fileSize))
				total += fileSize
			} else {
				err.Log(*l.PzSession, "Tried to get File Size from S3 File "+url+" but encountered an error; giving up on calculating input sizes")
				return 0
			}
		} else {
			pzsvc.LogInfo(*l.PzSession, fmt.Sprintf("Input file %s is not AWS; giving up on calculating input sizes", url))
			return 0
		}
	}
	return
}

func (l Loop) calculateDiskAndMemoryLimits(jobInput *pzsvc.InpStruct) (diskMB int, memoryMB int) {
	diskMB = defaultTaskDiskMB
	memoryMB = defaultTaskMemoryMB

	if inputSize := l.calculateAWSInputFileSizeMB(jobInput); inputSize > 0 {
		// Allocate space for the filesystem and executables (with some buffer), then add the image sizes
		diskMB = 4096 + (inputSize * 2)
		memoryMB = memoryMB + (inputSize * 5)
		pzsvc.LogInfo(*l.PzSession, fmt.Sprintf("Obtained S3 File Sizes for input files; will use Dynamic Disk Space of %d in Task container and Dynamic Memory Size of %d", diskMB, memoryMB))
	} else {
		pzsvc.LogInfo(*l.PzSession, fmt.Sprintf("Could not get the S3 File Sizes for input files. Will use the default Disk %d and Memory %d when running Task.", diskMB, memoryMB))
	}
	return
}
