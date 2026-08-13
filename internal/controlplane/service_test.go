package controlplane

import (
	"errors"
	"testing"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/scheduler"
	"github.com/sanketlengare/orchard/internal/store"
	"github.com/sanketlengare/orchard/internal/worker"
)

func TestNewService(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(nil, workerStore)
	if !errors.Is(err, ErrJobStoreRequired) {
		t.Fatalf("NewService() error = %v, want %v", err, ErrJobStoreRequired)
	}
	if service != nil {
		t.Fatalf("NewService() service = %v, want nil", service)
	}

	service, err = NewService(jobStore, nil)
	if !errors.Is(err, ErrWorkerStoreRequired) {
		t.Fatalf("NewService() error = %v, want %v", err, ErrWorkerStoreRequired)
	}
	if service != nil {
		t.Fatalf("NewService() service = %v, want nil", service)
	}

	service, err = NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if service == nil {
		t.Fatal("NewService() service = nil, want non-nil")
	}
	if service.jobStore != jobStore {
		t.Fatalf("NewService() job store = %v, want %v", service.jobStore, jobStore)
	}
	if service.workerStore != workerStore {
		t.Fatalf("NewService() worker store = %v, want %v", service.workerStore, workerStore)
	}
}

func TestSubmitJob(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	resources := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 4096,
	}

	submitted, err := service.SubmitJob(
		"job1",
		[]string{"echo", "hello"},
		resources,
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}
	if submitted == nil {
		t.Fatal("SubmitJob() job = nil, want non-nil")
	}

	stored, err := jobStore.Get("job1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored != submitted {
		t.Fatalf("stored job = %p, want submitted job %p", stored, submitted)
	}
}

func TestRegisterWorker(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	registered, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 8192,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}
	if registered == nil {
		t.Fatal("RegisterWorker() worker = nil, want non-nil")
	}

	stored, err := workerStore.Get("worker1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored != registered {
		t.Fatalf(
			"stored worker = %p, want registered worker %p",
			stored,
			registered,
		)
	}
}

func TestScheduleJob(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	resources := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 2048,
	}

	submitted, err := service.SubmitJob(
		"job1",
		[]string{"echo", "hello"},
		resources,
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	insufficient, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  1,
			MemoryMiB: 8192,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}

	suitable, err := service.RegisterWorker(
		"worker2",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 4096,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}

	insufficientBefore := insufficient.AvailableCapacity()

	selected, err := service.ScheduleJob("job1")
	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}
	if selected != suitable {
		t.Fatalf("ScheduleJob() worker = %v, want %v", selected, suitable)
	}

	if submitted.Status() != job.StatusRunning {
		t.Fatalf(
			"job status = %v, want %v",
			submitted.Status(),
			job.StatusRunning,
		)
	}

	wantAvailable := worker.Capacity{
		CPUCores:  2,
		MemoryMiB: 2048,
	}

	if available := suitable.AvailableCapacity(); available != wantAvailable {
		t.Fatalf(
			"selected worker capacity = %+v, want %+v",
			available,
			wantAvailable,
		)
	}

	if available := insufficient.AvailableCapacity(); available != insufficientBefore {
		t.Fatalf(
			"unselected worker capacity = %+v, want unchanged %+v",
			available,
			insufficientBefore,
		)
	}
}

func TestAssignedWorker(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	assigned, err := service.AssignedWorker("missing-job")
	if !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf(
			"AssignedWorker() error = %v, want %v",
			err,
			ErrAssignmentNotFound,
		)
	}
	if assigned != nil {
		t.Fatalf("AssignedWorker() worker = %v, want nil", assigned)
	}

	_, err = service.SubmitJob(
		"job1",
		[]string{"echo", "hello"},
		job.ResourceRequest{
			CPUCores:  2,
			MemoryMiB: 2048,
		},
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	registered, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 4096,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}

	selected, err := service.ScheduleJob("job1")
	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	assigned, err = service.AssignedWorker("job1")
	if err != nil {
		t.Fatalf("AssignedWorker() error = %v", err)
	}
	if assigned != selected {
		t.Fatalf(
			"AssignedWorker() worker = %v, want selected worker %v",
			assigned,
			selected,
		)
	}
	if assigned != registered {
		t.Fatalf(
			"AssignedWorker() worker = %v, want registered worker %v",
			assigned,
			registered,
		)
	}
}

func TestSuccessfulFinishJob(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	resources := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 2048,
	}

	submittedJob1, err := service.SubmitJob(
		"job1",
		[]string{"echo", "hello"},
		resources,
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	registered, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 4096,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}

	selected, err := service.ScheduleJob("job1")
	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}
	if selected != registered {
		t.Fatalf("ScheduleJob() worker = %v, want %v", selected, registered)
	}

	err = service.FinishJob("job1", job.StatusSucceeded)
	if err != nil {
		t.Fatalf("FinishJob() error = %v", err)
	}

	if submittedJob1.Status() != job.StatusSucceeded {
		t.Fatalf(
			"job status = %v, want %v",
			submittedJob1.Status(),
			job.StatusSucceeded,
		)
	}

	if registered.AvailableCapacity() != registered.TotalCapacity() {
		t.Fatalf(
			"available capacity = %+v, want %+v",
			registered.AvailableCapacity(),
			registered.TotalCapacity(),
		)
	}

	assigned, err := service.AssignedWorker("job1")
	if !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf(
			"AssignedWorker() error = %v, want %v",
			err,
			ErrAssignmentNotFound,
		)
	}
	if assigned != nil {
		t.Fatalf("AssignedWorker() worker = %v, want nil", assigned)
	}

	submittedJob2, err := service.SubmitJob(
		"job2",
		[]string{"echo", "hello"},
		resources,
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	if _, err := service.ScheduleJob("job2"); err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	err = service.FinishJob("job2", job.StatusFailed)
	if err != nil {
		t.Fatalf("FinishJob() error = %v", err)
	}

	if submittedJob2.Status() != job.StatusFailed {
		t.Fatalf(
			"job status = %v, want %v",
			submittedJob2.Status(),
			job.StatusFailed,
		)
	}
}

func TestInvalidJobFinish(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.SubmitJob(
		"job1",
		[]string{"echo", "hello"},
		job.ResourceRequest{
			CPUCores:  2,
			MemoryMiB: 2048,
		},
	)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	registered, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 4096,
		},
	)
	if err != nil {
		t.Fatalf("RegisterWorker() error = %v", err)
	}

	err = service.FinishJob("job1", job.StatusSucceeded)
	if !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf(
			"FinishJob() error = %v, want %v",
			err,
			ErrAssignmentNotFound,
		)
	}

	if _, err := service.ScheduleJob("job1"); err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	capacityBeforeFinish := registered.AvailableCapacity()

	err = service.FinishJob("job1", job.StatusPending)
	if !errors.Is(err, scheduler.ErrInvalidFinalStatus) {
		t.Fatalf(
			"FinishJob() error = %v, want %v",
			err,
			scheduler.ErrInvalidFinalStatus,
		)
	}

	assigned, err := service.AssignedWorker("job1")
	if err != nil {
		t.Fatalf("AssignedWorker() error = %v", err)
	}
	if assigned == nil {
		t.Fatal("AssignedWorker() worker = nil, want assigned worker")
	}
	if assigned != registered {
		t.Fatalf(
			"AssignedWorker() worker = %v, want %v",
			assigned,
			registered,
		)
	}

	if registered.AvailableCapacity() != capacityBeforeFinish {
		t.Fatalf(
			"available capacity = %+v, want unchanged %+v",
			registered.AvailableCapacity(),
			capacityBeforeFinish,
		)
	}
}
