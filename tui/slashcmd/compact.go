package slashcmd

func init() {
	Register(Command{"/compact", "Summarize and compact conversation history"})
}
