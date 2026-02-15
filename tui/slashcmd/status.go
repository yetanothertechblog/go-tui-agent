package slashcmd

func init() {
	Register(Command{"/status", "Show session status"})
}
