package exception

// ValidationError is what a person got wrong, by the field the request
// carries it in, raised past the request's own validation: from a rule,
// from an action, from anything that needs the database to know. It
// renders as the same 400 the request validator answers with, so a client
// reads both the same way.
//
//	return nil, exception.Validation("pipelineId", "No such pipeline.")
//
//	problems := exception.ValidationError{}
//	problems.Add("name", nameProblem(input.Name))
//	return graph, problems.Err()
type ValidationError map[string]string

// Validation is one field that is wrong.
func Validation(field, message string) ValidationError {
	return ValidationError{field: message}
}

func (ValidationError) Error() string { return "the request is not valid" }

// Add records a message under a field. An empty message is no error, so a
// check can hand its result straight in.
func (validation ValidationError) Add(field, message string) {
	if message != "" {
		validation[field] = message
	}
}

// Err is the error to return: nothing when nothing was recorded.
func (validation ValidationError) Err() error {
	if len(validation) == 0 {
		return nil
	}

	return validation
}
