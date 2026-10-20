package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	Version   = "8.0"
	PageURL   = "https://thatsillyman.win/resources/openbsd/syscall-table/"
	SourceURL = "https://cdn.openbsd.org/pub/OpenBSD/" + Version + "/sys.tar.gz"
	ManBase   = "https://man.openbsd.org/OpenBSD-" + Version + "/"
	Master    = "syscalls.master"
)

type Arg struct {
	Type string
	Name string
}

type Syscall struct {
	Number      int
	Name        string
	Types       []string
	Alias       string
	Comment     string
	Ret         string
	Entry       string
	Args        []Arg
	ArgsDisplay string
	Signature   string
	Condition   string
	HasProto    bool
	ManURL      string
	ManSection  string
}

// manSections overrides the default manual section (2) for specific syscalls.
var manSections = map[int]string{
	0: "9", // indirect syscall entry point, documented in syscall(9)
}

type TypeInfo struct {
	Name string
	Desc string
}

var typeLegend = []TypeInfo{
	{"STD", "always included in the system"},
	{"NOLOCK", "kernel lock is not acquired when called"},
	{"OBSOL", "obsolete, no longer included in the system"},
	{"UNIMPL", "unimplemented or reserved, not included in the system"},
	{"INDIR", "indirect entry, allows a really varargs prototype"},
	{"NOARGS", "included, but no syscall argument structure is defined"},
	{"NODEF", "included, but the syscall number is not defined"},
}

var (
	typeTokenRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	identRe     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	protoRe     = regexp.MustCompile(`^\{(.+)\}(\s+(\S+))?\s*$`)
	funcRe      = regexp.MustCompile(`sys_([A-Za-z0-9_]+)\s*\((.*)\)$`)
)

func main() {
	data, err := os.ReadFile(Master)
	if err != nil {
		log.Fatal(err)
	}
	syscalls := parse(string(data))
	if err := writeCSV("syscalls.csv", syscalls); err != nil {
		log.Fatal(err)
	}
	if err := writeHTML("index.html", syscalls); err != nil {
		log.Fatal(err)
	}

	counts := map[string]int{}
	for _, sc := range syscalls {
		for _, t := range sc.Types {
			counts[t]++
		}
	}
	var parts []string
	for t, n := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", t, n))
	}
	sort.Strings(parts)
	log.Printf("Generated index.html and syscalls.csv (%d syscalls; %s)", len(syscalls), strings.Join(parts, " "))
}

func parse(src string) []*Syscall {
	byNum := map[int]*Syscall{}
	var conds []string

	for _, line := range joinContinuations(strings.Split(src, "\n")) {
		t := strings.TrimSpace(line)
		switch {
		case t == "" || strings.HasPrefix(t, ";"), strings.HasPrefix(t, "#include"):
			continue
		case strings.HasPrefix(t, "#ifdef "):
			conds = append(conds, strings.TrimSpace(strings.TrimPrefix(t, "#ifdef ")))
			continue
		case strings.HasPrefix(t, "#ifndef "):
			conds = append(conds, "!"+strings.TrimSpace(strings.TrimPrefix(t, "#ifndef ")))
			continue
		case strings.HasPrefix(t, "#if "):
			conds = append(conds, strings.TrimSpace(strings.TrimPrefix(t, "#if ")))
			continue
		case t == "#else":
			if len(conds) > 0 {
				top := conds[len(conds)-1]
				if strings.HasPrefix(top, "!") {
					top = top[1:]
				} else {
					top = "!" + top
				}
				conds[len(conds)-1] = top
			}
			continue
		case t == "#endif":
			if len(conds) > 0 {
				conds = conds[:len(conds)-1]
			}
			continue
		case strings.HasPrefix(t, "#"):
			continue
		}

		sc, ok := parseEntry(t, strings.Join(conds, " && "))
		if !ok {
			continue
		}
		if old, exists := byNum[sc.Number]; exists {
			if score(sc) > score(old) {
				byNum[sc.Number] = sc
			}
		} else {
			byNum[sc.Number] = sc
		}
	}

	list := make([]*Syscall, 0, len(byNum))
	for _, sc := range byNum {
		section := "2"
		if s, ok := manSections[sc.Number]; ok {
			section = s
		}
		sc.ManSection = section
		if sc.Name != "" {
			sc.ManURL = ManBase + sc.Name + "." + section
		}
		list = append(list, sc)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Number < list[j].Number })
	return list
}

func score(sc *Syscall) int {
	switch {
	case sc.HasProto:
		return 10
	case sc.Name != "":
		return 5
	default:
		return 0
	}
}

func joinContinuations(lines []string) []string {
	var out []string
	var buf string
	for _, l := range lines {
		if buf != "" {
			buf += " " + strings.TrimSpace(l)
		} else {
			buf = l
		}
		if strings.HasSuffix(strings.TrimRight(buf, " \t"), "\\") {
			buf = strings.TrimSuffix(strings.TrimRight(buf, " \t"), "\\")
			continue
		}
		out = append(out, buf)
		buf = ""
	}
	if buf != "" {
		out = append(out, buf)
	}
	return out
}

func parseEntry(line, cond string) (*Syscall, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, false
	}
	num, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, false
	}

	i := 1
	var types []string
	for i < len(fields) && typeTokenRe.MatchString(fields[i]) {
		types = append(types, fields[i])
		i++
	}
	if len(types) == 0 {
		return nil, false
	}
	rest := strings.Join(fields[i:], " ")
	sc := &Syscall{Number: num, Types: types, Condition: cond}

	if strings.HasPrefix(rest, "{") {
		m := protoRe.FindStringSubmatch(rest)
		if m == nil {
			return nil, false
		}
		inner := strings.TrimSpace(m[1])
		inner = strings.TrimSpace(strings.TrimSuffix(inner, ";"))
		loc := funcRe.FindStringSubmatchIndex(inner)
		if loc == nil {
			log.Printf("unparsed prototype for syscall %d: %s", num, inner)
			return nil, false
		}
		sc.HasProto = true
		sc.Ret = strings.TrimSpace(inner[:loc[0]])
		sc.Name = inner[loc[2]:loc[3]]
		sc.Entry = "sys_" + sc.Name
		sc.ArgsDisplay = strings.TrimSpace(inner[loc[4]:loc[5]])
		sc.Signature = sc.Entry + "(" + sc.ArgsDisplay + ")"
		if m[3] != "" {
			sc.Alias = m[3]
		}
		sc.Args = parseArgs(sc.ArgsDisplay)
		return sc, true
	}

	sc.Comment = rest
	if fields := strings.Fields(rest); len(fields) == 1 && identRe.MatchString(fields[0]) {
		sc.Name = strings.TrimPrefix(fields[0], "sys_")
	}
	sc.Signature = rest
	return sc, true
}

func parseArgs(s string) []Arg {
	s = strings.TrimSpace(s)
	if s == "" || s == "void" {
		return nil
	}
	var args []Arg
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		f := strings.Fields(part)
		if len(f) == 1 {
			args = append(args, Arg{Type: f[0]})
			continue
		}
		name := f[len(f)-1]
		typ := strings.Join(f[:len(f)-1], " ")
		if strings.HasPrefix(name, "*") {
			typ = strings.TrimSpace(typ + " " + name[:strings.Index(name, "*")+1])
			name = name[strings.Index(name, "*")+1:]
		}
		args = append(args, Arg{Type: typ, Name: name})
	}
	return args
}

func writeCSV(path string, syscalls []*Syscall) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"number", "name", "types", "alias", "signature", "condition", "man_url"}); err != nil {
		return err
	}
	for _, sc := range syscalls {
		if err := w.Write([]string{
			strconv.Itoa(sc.Number),
			sc.Name,
			strings.Join(sc.Types, " "),
			sc.Alias,
			sc.Signature,
			sc.Condition,
			sc.ManURL,
		}); err != nil {
			return err
		}
	}
	return w.Error()
}

func writeHTML(path string, syscalls []*Syscall) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("index.html.tmpl").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).ParseFiles("index.html.tmpl")
	if err != nil {
		return err
	}
	return tmpl.Execute(f, map[string]interface{}{
		"Title":      "Searchable OpenBSD Syscall Table for OpenBSD " + Version,
		"Version":    Version,
		"PageURL":    PageURL,
		"SourceURL":  SourceURL,
		"TypeLegend": typeLegend,
		"Syscalls":   syscalls,
	})
}
