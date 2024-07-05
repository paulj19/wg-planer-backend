package main

import (
	"reflect"
	"testing"
)

var f Floor

func Test_findTask(t *testing.T) {
	t.Run("should find task", func(t *testing.T) {
		task := findTask(f.Tasks, f.Tasks[len(f.Tasks)-1].Id)
		if !reflect.DeepEqual(task, f.Tasks[len(f.Tasks)-1]) {
			t.Errorf("task not found: got %v want %v", task, f.Tasks[len(f.Tasks)-1])
		}
	})
	t.Run("should not find task", func(t *testing.T) {
		task := findTask(f.Tasks, "9999")
		var taskNil Task
		if !reflect.DeepEqual(task, taskNil) {
			t.Error("findTask did not return zero value task")
		}
	})
}

func Test_nextAssignee(t *testing.T) {
	t.Run("should find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextRoom, err := nextAssignee(f, f.Tasks[0])
		if err != nil {
			t.Fatal(err)
		}
		if nextRoom.Id != 2 {
			t.Errorf("next assignee not found: got %v want %v", nextRoom.Id, f.Tasks[1].AssignedTo)
		}
	})
	t.Run("should not assign to non-avail resident", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    4,
					Order: 3,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextRoom, err := nextAssignee(f, f.Tasks[0])
		if err != nil {
			t.Fatal(err)
		}
		if nextRoom.Id != 4 {
			t.Errorf("next assignee not found: got %v want %v", nextRoom.Id, f.Tasks[1].AssignedTo)
		}
	})
	t.Run("should not find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    4,
					Order: 3,
					Resident: Resident{
						Available: false,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextAss, err := nextAssignee(f, f.Tasks[0])
		if err == nil || err.Error() != "No next assignee available" || !reflect.DeepEqual(nextAss, Room{}) {
			t.Errorf("no avail residents, nextAss should be emtpy room with an error")
		}
	})
	t.Run("should not find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextAss, err := nextAssignee(f, f.Tasks[0])
		if err == nil || err.Error() != "No next assignee available" || !reflect.DeepEqual(nextAss, Room{}) {
			t.Errorf("no avail residents, nextAss should be emtpy room with an error")
		}
	})
}
