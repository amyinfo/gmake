package shuffle

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/amyinfo/gmake/pkg/config"
	"github.com/amyinfo/gmake/pkg/types"
)

type shuffleMode int

const (
	smNone     shuffleMode = iota
	smRandom
	smReverse
	smIdentity
)

type shuffleConfig struct {
	mode     shuffleMode
	seed     int64
	shuffler func(a []interface{})
	strval   string
}

var config_ shuffleConfig
var rng = rand.New(rand.NewSource(0)) // replaced on random mode init

func GetMode() string {
	if config_.strval == "" {
		return ""
	}
	return config_.strval
}

func SetMode(cmdarg string) {
	switch strings.ToLower(cmdarg) {
	case "reverse":
		config_.mode = smReverse
		config_.shuffler = reverseShuffleArray
		config_.strval = "reverse"
	case "identity":
		config_.mode = smIdentity
		config_.shuffler = identityShuffleArray
		config_.strval = "identity"
	case "none":
		config_.mode = smNone
		config_.shuffler = nil
		config_.strval = ""
	default:
		if strings.ToLower(cmdarg) == "random" {
			config_.seed = rng.Int63()
		} else {
			seed, err := strconv.ParseInt(cmdarg, 10, 64)
			if err != nil {
				return
			}
			config_.seed = seed
		}
		config_.mode = smRandom
		config_.shuffler = randomShuffleArray
		config_.strval = strconv.FormatInt(config_.seed, 10)
	}
}

func randomShuffleArray(a []interface{}) {
	for i := len(a) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func reverseShuffleArray(a []interface{}) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

func identityShuffleArray(a []interface{}) {
}

func shuffleDeps(deps *types.Dep) {
	var ndeps int
	for dep := deps; dep != nil; dep = dep.Next {
		if dep.WaitHere {
			return
		}
		ndeps++
	}

	if ndeps == 0 {
		return
	}

	da := make([]interface{}, ndeps)
	var i int
	for dep := deps; dep != nil; dep = dep.Next {
		da[i] = dep
		i++
	}

	config_.shuffler(da)

	i = 0
	for dep := deps; dep != nil; dep = dep.Next {
		dep.Shuf = da[i].(*types.Dep)
		i++
	}
}

func shuffleFileDepsRecursive(f *types.File) {
	if f == nil {
		return
	}

	if f.WasShuffled {
		return
	}
	f.WasShuffled = true

	shuffleDeps(f.Deps)

	for dep := f.Deps; dep != nil; dep = dep.Next {
		shuffleFileDepsRecursive(dep.File)
	}
}

func ShuffleDepsRecursive(deps *types.Dep) {
	if config_.mode == smNone {
		return
	}

	if config.NotParallel {
		return
	}

	if config_.mode == smRandom {
		rng = rand.New(rand.NewSource(config_.seed))
	}

	shuffleDeps(deps)

	for dep := deps; dep != nil; dep = dep.Next {
		shuffleFileDepsRecursive(dep.File)
	}
}

func Init() {
	if config.ShuffleMode != "" {
		SetMode(config.ShuffleMode)
	}
}
