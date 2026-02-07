package llm

func CallLLMMock(_ []Message, _ []Tool, responseText string) (*Delta, error) {
	return &Delta{
		Role:    "assistant",
		Content: responseText,
	}, nil
}
