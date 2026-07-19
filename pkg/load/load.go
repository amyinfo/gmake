package load

import (
	"plugin"
	"strings"

	"github.com/amyinfo/gmake/pkg/debug"
	"github.com/amyinfo/gmake/pkg/file"
	"github.com/amyinfo/gmake/pkg/misc"
	"github.com/amyinfo/gmake/pkg/strcache"
	"github.com/amyinfo/gmake/pkg/types"
	"github.com/amyinfo/gmake/pkg/variable"
)

type loadEntry struct {
	Name string
	Plg  *plugin.Plugin
}

var loadedSyms []loadEntry

type loadFunc func(*types.Floc) int

func loadObject(flocp *types.Floc, noerror int, ldname, symname string) loadFunc {
	for _, entry := range loadedSyms {
		if entry.Name == ldname {
			sym, err := entry.Plg.Lookup(symname)
			if err == nil {
				if f, ok := sym.(func(*types.Floc) int); ok {
					return f
				}
			}
		}
	}

	plg, err := plugin.Open(ldname)
	if err != nil {
		if noerror == 0 {
			if flocp != nil {
				debug.DB(debug.Basic, "%s\n", err.Error())
			}
		}
		return nil
	}

	debug.DB(debug.Verbose, "Loaded shared object %s\n", ldname)

	_, err = plg.Lookup("plugin_is_GPL_compatible")
	if err != nil {
		return nil
	}

	sym, err := plg.Lookup(symname)
	if err != nil {
		return nil
	}

	f, ok := sym.(func(*types.Floc) int)
	if !ok {
		return nil
	}

	loadedSyms = append(loadedSyms, loadEntry{Name: misc.Xstrdup(ldname), Plg: plg})
	return f
}

func LoadFile(flocp *types.Floc, file_ *types.File, noerror int) int {
	ldname := file_.Name
	nmlen := len(ldname)
	buf := make([]byte, nmlen+len("_gmk_setup")+1)
	var symname string

	fp := strings.IndexByte(ldname, '(')
	if fp >= 0 {
		ep := strings.IndexByte(ldname[fp+1:], ')')
		if ep >= 0 && fp+1+ep+1 == len(ldname) {
			l := fp
			copy(buf, ldname[:l])
			buf[l] = 0
			ldname = string(buf[:l])
			nmlen = l
			symname = ldname[fp+1 : fp+1+ep]
		}
	}

	ldname = strcache.Add(ldname)

	tgt := file.LookupFile(ldname)
	if tgt != nil && tgt.Loaded {
		return -1
	}

	if symname == "" {
		p := buf[:0]
		fp = strings.LastIndexByte(ldname, '/')
		if fp < 0 {
			fp = 0
		} else {
			fp++
		}
		for i := fp; i < len(ldname); i++ {
			c := ldname[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
				p = append(p, c)
			}
		}
		p = append(p, "_gmk_setup"...)
		symname = string(p)
	}

	debug.DB(debug.Verbose, "Loading symbol %s from %s\n", symname, ldname)

	symp := loadObject(flocp, noerror, ldname, symname)
	if symp == nil {
		return 0
	}

	r := symp(flocp)

	if r != 0 {
		variable.DoVariableDefinition(flocp, ".LOADED", ldname, types.OriginFile, types.FlavorAppend, 0)
	}

	return r
}

func UnloadFile(name string) int {
	for i, entry := range loadedSyms {
		if entry.Name == name && entry.Plg != nil {
			debug.DB(debug.Verbose, "Unloading shared object %s\n", name)
			loadedSyms = append(loadedSyms[:i], loadedSyms[i+1:]...)
			return 0
		}
	}
	return 0
}
