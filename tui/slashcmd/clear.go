package slashcmd

func init() {
	Register(Command{"/clear", "Clear conversation history"})
}
