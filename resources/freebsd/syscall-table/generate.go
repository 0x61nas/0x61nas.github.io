package main

import (
	"encoding/csv"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	Version   = "15.0"
	ManPath   = "FreeBSD+" + Version + "-RELEASE"
	PageURL   = "https://thatsillyman.win/resources/freebsd/syscall-table/"
	SourceURL = "https://cgit.freebsd.org/src/plain/sys/kern/syscalls.master?h=releng/" + Version
	ManBase   = "https://man.freebsd.org/cgi/man.cgi?query="
	Master    = "syscalls.master"
)

type Arg struct {
	Type string
	Name string
}

type Syscall struct {
	Number      string
	Name        string
	Types       []string
	Audit       string
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
}

type TypeInfo struct {
	Name string
	Desc string
}

var typeLegend = []TypeInfo{
	{"STD", "always included"},
	{"CAPENABLED", "allowed in capability mode"},
	{"COMPAT", "included only on COMPAT kernels"},
	{"COMPAT4-14", "included only for the matching COMPAT_FREEBSD<n> ABI"},
	{"OBSOL", "obsolete, not included in the system"},
	{"RESERVED", "reserved for local or vendor use"},
	{"UNIMPL", "not implemented, placeholder only"},
	{"NOSTD", "implemented as a loadable module"},
	{"NODEF", "entry only, no structure or prototype is generated"},
	{"NOARGS", "like STD, but no argument structure is generated"},
	{"NOPROTO", "like STD, but no prototype is generated"},
	{"NOLIB", "no stubs are generated in libc or libsys"},
	{"NOTSTATIC", "the syscall is loadable"},
	{"NORETURN", "the syscall does not return"},
	{"SYSMUX", "syscall multiplexer, handled in MD syscall code"},
}

type archConfig struct {
	Name       string
	Arch       string
	Title      string
	PageURL    string
	Registers  []string
	ReturnReg  string
	SyscallReg string
	ABIRefName string
	ABIRefURL  string
	OutDir     string
}

var (
	typeTokenRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*(\|[A-Z][A-Z0-9_]*)*$`)
	salRe       = regexp.MustCompile(`\b_(?:In|Out|Inout|Contains|Post|Pre|Field|Struct|Reserved)[A-Za-z0-9_]*_(?:\([^)]*\))?`)
	funcRe      = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$`)
)

func main() {
	data, err := os.ReadFile(Master)
	if err != nil {
		log.Fatal(err)
	}
	syscalls := parse(string(data))

	configs := []archConfig{
		{
			Name:       "x86_64",
			Arch:       "x86_64",
			Title:      "Searchable FreeBSD Syscall Table for x86_64",
			PageURL:    PageURL,
			Registers:  []string{"rdi", "rsi", "rdx", "r10", "r8", "r9"},
			ReturnReg:  "%rax",
			SyscallReg: "%rax",
			ABIRefName: "x86-64.net syscall ABI",
			ABIRefURL:  "https://x86-64.net/abi/syscalls/",
			OutDir:     "x86_64",
		},
		{
			Name:       "arm64",
			Arch:       "ARM64",
			Title:      "Searchable FreeBSD Syscall Table for ARM64",
			PageURL:    PageURL + "arm64/",
			Registers:  []string{"x0", "x1", "x2", "x3", "x4", "x5"},
			ReturnReg:  "%x0",
			SyscallReg: "%x8",
			ABIRefName: "Arm 64-bit Procedure Call Standard (AAPCS64)",
			ABIRefURL:  "https://developer.arm.com/documentation/102374/0102/Procedure-Call-Standard",
			OutDir:     "arm64",
		},
		{
			Name:       "riscv64",
			Arch:       "RISC-V",
			Title:      "Searchable FreeBSD Syscall Table for RISC-V",
			PageURL:    PageURL + "riscv64/",
			Registers:  []string{"a0", "a1", "a2", "a3", "a4", "a5"},
			ReturnReg:  "%a0",
			SyscallReg: "%a7",
			ABIRefName: "RISC-V Register Convention (psABI)",
			ABIRefURL:  "https://docs.riscv.org/reference/abi/v1.0/riscv-cc-register-convention.html",
			OutDir:     "riscv64",
		},
	}

	if err := writeCSV("syscalls.csv", syscalls); err != nil {
		log.Printf("CSV error: %v", err)
	}

	var wg sync.WaitGroup
	for _, conf := range configs {
		wg.Add(1)
		go func(c archConfig) {
			defer wg.Done()
			generatePage(c, syscalls)
		}(conf)
	}
	wg.Wait()

	counts := map[string]int{}
	for _, sc := range syscalls {
		for _, t := range sc.Types {
			counts[t]++
		}
	}
	log.Printf("Generated %d architecture pages (%d syscalls)", len(configs), len(syscalls))
}

func generatePage(conf archConfig, syscalls []*Syscall) {
	if err := os.MkdirAll(conf.OutDir, 0o755); err != nil {
		log.Printf("[%s] mkdir error: %v", conf.Name, err)
		return
	}
	csvLink := conf.OutDir + "/syscalls.csv"
	if err := os.Remove(csvLink); err != nil && !os.IsNotExist(err) {
		log.Printf("[%s] CSV remove error: %v", conf.Name, err)
	}
	if err := os.Symlink("../syscalls.csv", csvLink); err != nil {
		log.Printf("[%s] CSV symlink error: %v", conf.Name, err)
	}

	tmpl, err := template.New("index.html.tmpl").ParseFiles("index.html.tmpl")
	if err != nil {
		log.Printf("[%s] Template error: %v", conf.Name, err)
		return
	}
	f, err := os.Create(conf.OutDir + "/index.html")
	if err != nil {
		log.Printf("[%s] File error: %v", conf.Name, err)
		return
	}
	defer f.Close()

	err = tmpl.Execute(f, map[string]interface{}{
		"Title":      conf.Title,
		"Arch":       conf.Arch,
		"PageURL":    conf.PageURL,
		"Version":    Version,
		"SourceURL":  SourceURL,
		"ABIRefName": conf.ABIRefName,
		"ABIRefURL":  conf.ABIRefURL,
		"TypeLegend": typeLegend,
		"Syscalls":   syscalls,
		"Registers":  conf.Registers,
		"ReturnReg":  conf.ReturnReg,
		"SyscallReg": conf.SyscallReg,
	})
	if err != nil {
		log.Printf("[%s] Execute error: %v", conf.Name, err)
		return
	}
	log.Printf("Generated %s/index.html", conf.OutDir)
}

func parse(src string) []*Syscall {
	var syscalls []*Syscall
	lines := strings.Split(src, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		sc, consumed, ok := parseEntry(lines, i)
		if !ok {
			continue
		}
		i += consumed
		syscalls = append(syscalls, sc)
	}

	sort.SliceStable(syscalls, func(a, b int) bool {
		return firstNumber(syscalls[a].Number) < firstNumber(syscalls[b].Number)
	})
	return syscalls
}

func firstNumber(s string) int {
	s = strings.SplitN(s, "-", 2)[0]
	n, _ := strconv.Atoi(s)
	return n
}

// parseEntry parses one syscall entry starting at lines[i], returning the
// syscall and the number of extra lines it consumed (for multi-line prototypes).
func parseEntry(lines []string, i int) (*Syscall, int, bool) {
	line := lines[i]
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, 0, false
	}
	if !isNumberSpec(fields[0]) {
		return nil, 0, false
	}
	if !strings.HasPrefix(fields[1], "AUE_") {
		return nil, 0, false
	}
	if !typeTokenRe.MatchString(fields[2]) {
		return nil, 0, false
	}

	sc := &Syscall{
		Number: fields[0],
		Audit:  fields[1],
		Types:  strings.Split(fields[2], "|"),
	}
	sc.Condition = conditionFor(sc.Types)

	rest := strings.Join(fields[3:], " ")

	if strings.HasPrefix(rest, "{") || strings.Contains(rest, "{") {
		// Multi-line prototype: accumulate until closing brace.
		buf := rest
		consumed := 0
		j := i
		for !strings.Contains(buf, "}") && j+1 < len(lines) {
			j++
			consumed++
			buf += " " + strings.TrimSpace(lines[j])
		}
		proto := extractProto(buf)
		if proto != "" {
			sc.HasProto = true
			sc.Ret, sc.Entry, sc.ArgsDisplay, sc.Args = parseProto(proto)
			if sc.Entry != "" {
				sc.Name = strings.TrimPrefix(sc.Entry, "sys_")
				sc.Signature = sc.Entry + "(" + sc.ArgsDisplay + ")"
			}
			if strings.Contains(buf, "}") {
				tail := strings.TrimSpace(buf[strings.Index(buf, "}")+1:])
				if tail != "" {
					sc.Alias = tail
					if sc.Name == "" {
						sc.Name = tail
					}
				}
			}
		}
		if sc.Name != "" {
			sc.ManURL = ManBase + sc.Name + "&sektion=2&manpath=" + ManPath
		}
		return sc, consumed, true
	}

	// No prototype: OBSOL/UNIMPL/RESERVED/NODEF-alt rows with trailing names.
	// Format: number AUE type [altname [alttag [altrtyp]]]
	sc.Comment = rest
	tokens := strings.Fields(rest)
	if len(tokens) > 0 {
		sc.Name = tokens[0]
		if len(tokens) >= 2 {
			sc.Alias = tokens[1]
		}
	}
	if sc.Name != "" {
		sc.ManURL = ManBase + sc.Name + "&sektion=2&manpath=" + ManPath
	}
	return sc, 0, true
}

func isNumberSpec(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.SplitN(s, "-", 2)
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return len(parts) == 1 || len(parts) == 2
}

func conditionFor(types []string) string {
	var conds []string
	for _, t := range types {
		switch {
		case t == "COMPAT":
			conds = append(conds, "COMPAT_FREEBSD")
		case strings.HasPrefix(t, "COMPAT") && t != "COMPAT":
			conds = append(conds, "COMPAT_FREEBSD"+strings.TrimPrefix(t, "COMPAT"))
		case t == "NOSTD":
			conds = append(conds, "loadable kernel module")
		}
	}
	return strings.Join(conds, " && ")
}

// extractProto returns the text between the first '{' and its matching '}'.
func extractProto(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

func parseProto(proto string) (ret, entry, argsDisplay string, args []Arg) {
	proto = strings.Join(strings.Fields(proto), " ")
	proto = strings.TrimSpace(strings.TrimSuffix(proto, ";"))
	m := funcRe.FindStringSubmatch(proto)
	if m == nil {
		return "", "", "", nil
	}
	name := m[1]
	params := m[2]
	// Return type is everything before the function name (after stripping SAL).
	idx := strings.LastIndex(proto, name)
	retRaw := strings.TrimSpace(proto[:idx])
	retRaw = salRe.ReplaceAllString(retRaw, "")
	ret = strings.Join(strings.Fields(retRaw), " ")

	params = salRe.ReplaceAllString(params, "")
	argsDisplay = strings.Join(strings.Fields(params), " ")
	args = parseArgs(params)
	return ret, "sys_" + name, argsDisplay, args
}

func parseArgs(s string) []Arg {
	s = strings.TrimSpace(s)
	if s == "" || s == "void" {
		return nil
	}
	var args []Arg
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" || part == "..." {
			continue
		}
		f := strings.Fields(part)
		if len(f) == 1 {
			args = append(args, Arg{Type: f[0]})
			continue
		}
		name := f[len(f)-1]
		typ := strings.Join(f[:len(f)-1], " ")
		if i := strings.Index(name, "*"); i >= 0 {
			typ = strings.TrimSpace(typ + " " + name[:i+1])
			name = name[i+1:]
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
	if err := w.Write([]string{"number", "name", "types", "audit", "alias", "signature", "condition", "man_url"}); err != nil {
		return err
	}
	for _, sc := range syscalls {
		if err := w.Write([]string{
			sc.Number,
			sc.Name,
			strings.Join(sc.Types, "|"),
			sc.Audit,
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
