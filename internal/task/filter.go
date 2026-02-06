package task

type Filter struct {
	Status   *Status
	Priority *Priority
	Tags     []string
	Query    string
}

func (f Filter) Match(t Task) bool {
	if f.Status != nil && t.Status != *f.Status {
		return false
	}
	if f.Priority != nil && t.Priority != *f.Priority {
		return false
	}
	return true
}

func (f Filter) Apply(tasks []Task) []Task {
	if f.Status == nil && f.Priority == nil && len(f.Tags) == 0 && f.Query == "" {
		return tasks
	}
	var result []Task
	for _, t := range tasks {
		if f.Match(t) {
			result = append(result, t)
		}
	}
	return result
}
