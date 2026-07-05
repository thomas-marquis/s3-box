package settings

// State defines the interface for entity states
type State interface {
	// CanRegister returns true if registration is permitted in this state
	CanRegister() bool
	// CanLoad returns true if loading is permitted in this state
	CanLoad() bool
	// CanSave returns true if saving is permitted in this state
	CanSave() bool
	// CanWrite returns true if writing is permitted in this state
	CanWrite() bool
	// String returns the state name for debugging
	String() string
}

// IdleState is the initial state where all operations are permitted
type IdleState struct{}

func (IdleState) CanRegister() bool { return true }
func (IdleState) CanLoad() bool     { return true }
func (IdleState) CanSave() bool     { return true }
func (IdleState) CanWrite() bool    { return true }
func (IdleState) String() string    { return "idle" }

// LoadingState is the state during load operation
type LoadingState struct{}

func (LoadingState) CanRegister() bool { return false }
func (LoadingState) CanLoad() bool     { return false }
func (LoadingState) CanSave() bool     { return false }
func (LoadingState) CanWrite() bool    { return false }
func (LoadingState) String() string    { return "loading" }

// SavingState is the state during save operation
type SavingState struct{}

func (SavingState) CanRegister() bool { return true }
func (SavingState) CanLoad() bool     { return false }
func (SavingState) CanSave() bool     { return false }
func (SavingState) CanWrite() bool    { return true }
func (SavingState) String() string    { return "saving" }
