package types

// Floc represents the location of elements read from makefiles.
// Port of struct floc from makeint.h (lines 529-534)
type Floc struct {
	Filenm string
	Lineno uint64
	Offset uint64
}

// Make Floc comparable as a nil pointer check
var NILF *Floc = nil

// VariableOrigin codes for variable definition origin.
// Port of enum variable_origin from variable.h (lines 23-33)
type VariableOrigin int

const (
	OriginDefault     VariableOrigin = iota // from default set
	OriginEnv                               // from environment
	OriginFile                              // from a makefile
	OriginEnvOverride                       // from environment with -e
	OriginCommand                           // from user command line
	OriginOverride                          // from override directive
	OriginAutomatic                         // automatic variable, cannot be set
	OriginInvalid                           // core dump time
)

// VariableFlavor for variable definition flavors.
// Port of enum variable_flavor from variable.h (lines 36-46)
type VariableFlavor int

const (
	FlavorBogus       VariableFlavor = iota
	FlavorSimple                     // := or ::=
	FlavorRecursive                  // =
	FlavorExpand                     // POSIX :::=
	FlavorAppend                     // +=
	FlavorConditional                // ?=
	FlavorShell                      // !=
	FlavorAppendValue                // append unexpanded value
)

// VariableExport controls variable exporting.
// Port of enum variable_export from variable.h (lines 47-53)
type VariableExport int

const (
	ExportDefault  VariableExport = iota // decide in target_environment
	ExportYes                            // export this variable
	ExportNo                             // don't export
	ExportIfSet                          // export if non-default value
)

// UpdateStatus for file update attempts.
// Port of enum update_status from filedef.h (lines 69-75)
type UpdateStatus int

const (
	UpdateSuccess  UpdateStatus = 0 // successfully updated
	UpdateNone     UpdateStatus = 1 // no attempt yet
	UpdateQuestion UpdateStatus = 2 // needs update (-q)
	UpdateFailed   UpdateStatus = 3 // update failed
)

// CmdState for command execution state.
// Port of enum cmd_state from filedef.h (lines 76-82)
type CmdState int

const (
	CmdNotStarted  CmdState = 0 // not yet started
	CmdDepsRunning CmdState = 1 // dep commands running
	CmdRunning     CmdState = 2 // commands running
	CmdFinished    CmdState = 3 // commands finished
)

// Read makefile flags.
// Port of RM_* flags from dep.h (lines 33-37)
const (
	RMNoFlag         = 0
	RMNoDefaultGoal  = 1 << 0
	RMIncluded       = 1 << 1
	RMSearchPath     = 1 << 1 // alias
	RMDontcare       = 1 << 2
	RMNoTilde        = 1 << 3
)

// Parse file seq flags.
// Port of PARSEFS_* from dep.h (lines 78-85)
const (
	ParsefsNone    = 0x0000
	ParsefsNostrip = 0x0001
	ParsefsNoar    = 0x0002
	ParsefsNoglob  = 0x0004
	ParsefsExists  = 0x0008
	ParsefsNocache = 0x0010
	ParsefsOneword = 0x0020
	ParsefsWait    = 0x0040
)

// Command line flags.
// Port of COMMANDS_* from commands.h (lines 33-35)
const (
	CmdRecurse = 1 // + or $(MAKE)
	CmdSilent  = 2 // @
	CmdNoerror = 4 // -
)
