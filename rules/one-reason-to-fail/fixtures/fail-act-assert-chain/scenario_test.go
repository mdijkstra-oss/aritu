package scenario

import "testing"

func TestStackHandlesPushingPoppingAndDraining(t *testing.T) {
	var stack Stack

	stack.Push(1)
	if stack.Len() != 1 {
		t.Fatalf("Len after one push = %d; want 1", stack.Len())
	}

	stack.Push(2)
	if stack.Len() != 2 {
		t.Fatalf("Len after two pushes = %d; want 2", stack.Len())
	}

	top, hasValue := stack.Pop()
	if !hasValue {
		t.Fatal("Pop on a stack holding two items reported it empty")
	}
	if top != 2 {
		t.Errorf("Pop = %d; want 2", top)
	}

	stack.Push(3)
	if stack.Len() != 2 {
		t.Errorf("Len after popping one and pushing one = %d; want 2", stack.Len())
	}

	stack.Pop()
	stack.Pop()
	if _, hasValue := stack.Pop(); hasValue {
		t.Error("Pop on a drained stack reported a value; want none")
	}
}
