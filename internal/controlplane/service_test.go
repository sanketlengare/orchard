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

	invalid, err := service.SubmitJob(
		"   ",
		[]string{"echo", "hello"},
		resources,
	)
	if !errors.Is(err, job.ErrIDRequired) {
		t.Fatalf("SubmitJob() error = %v, want %v", err, job.ErrIDRequired)
	}
	if invalid != nil {
		t.Fatalf("SubmitJob() job = %v, want nil", invalid)
	}

	duplicate, err := service.SubmitJob(
		"job1",
		[]string{"printf", "different"},
		job.ResourceRequest{
			CPUCores:  1,
			MemoryMiB: 1024,
		},
	)
	if !errors.Is(err, store.ErrJobAlreadyExists) {
		t.Fatalf(
			"SubmitJob() error = %v, want %v",
			err,
			store.ErrJobAlreadyExists,
		)
	}
	if duplicate != nil {
		t.Fatalf("SubmitJob() job = %v, want nil", duplicate)
	}

	stored, err = jobStore.Get("job1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored != submitted {
		t.Fatalf("stored job = %p, want original job %p", stored, submitted)
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

	invalid, err := service.RegisterWorker(
		"   ",
		worker.Capacity{
			CPUCores:  4,
			MemoryMiB: 8192,
		},
	)
	if !errors.Is(err, worker.ErrIDRequired) {
		t.Fatalf(
			"RegisterWorker() error = %v, want %v",
			err,
			worker.ErrIDRequired,
		)
	}
	if invalid != nil {
		t.Fatalf("RegisterWorker() worker = %v, want nil", invalid)
	}

	duplicate, err := service.RegisterWorker(
		"worker1",
		worker.Capacity{
			CPUCores:  1,
			MemoryMiB: 1024,
		},
	)
	if !errors.Is(err, store.ErrWorkerAlreadyExists) {
		t.Fatalf(
			"RegisterWorker() error = %v, want %v",
			err,
			store.ErrWorkerAlreadyExists,
		)
	}
	if duplicate != nil {
		t.Fatalf("RegisterWorker() worker = %v, want nil", duplicate)
	}

	stored, err = workerStore.Get("worker1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored != registered {
		t.Fatalf(
			"stored worker = %p, want original worker %p",
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

	capacityAfterFirstSchedule := suitable.AvailableCapacity()

	selected, err = service.ScheduleJob("job1")
	if !errors.Is(err, scheduler.ErrJobNotPending) {
		t.Fatalf(
			"second ScheduleJob() error = %v, want %v",
			err,
			scheduler.ErrJobNotPending,
		)
	}
	if selected != nil {
		t.Fatalf("second ScheduleJob() worker = %v, want nil", selected)
	}

	if available := suitable.AvailableCapacity(); available != capacityAfterFirstSchedule {
		t.Fatalf(
			"capacity after second schedule = %+v, want unchanged %+v",
			available,
			capacityAfterFirstSchedule,
		)
	}
}
