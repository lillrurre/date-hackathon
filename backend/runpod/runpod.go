package runpod

type Input struct {
	Prompt string `json:"prompt"`
	System string `json:"system"`
}

type Output struct {
	Done               bool   `json:"done"`
	Response           string `json:"response"`
	Model              string `json:"model"`
	CreatedAt          string `json:"-"`
	EvalCount          int    `json:"-"`
	EvalDuration       int    `json:"-"`
	LoadDuration       int    `json:"-"`
	PromptEvalCount    int    `json:"-"`
	PromptEvalDuration int    `json:"-"`
	TotalDuration      int    `json:"-"`
	Context            []int  `json:"-"`
}

type Job struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Input         Input  `json:"input"`
	Output        Output `json:"output"`
	DelayTime     int    `json:"-"`
	ExecutionTime int    `json:"-"`
}
