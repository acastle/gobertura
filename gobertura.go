package main

import (
  "encoding/xml"
  "golang.org/x/tools/cover"
  "fmt"
  "os"
  "path/filepath"
  "time"
  "flag"
)

var (
  coverprof  = flag.String("coverprofile", "", "Required: The go cover profile to use for conversion")
)

func usage() {
  cmd := os.Args[0]
  fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", cmd)
  flag.PrintDefaults()
}

func profileCoverage(p *cover.Profile) (int64, int64) {
	var total, covered int64
	for _, b := range p.Blocks {
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		}
	}
	if total == 0 {
		return 0, 0
	}
	return total, covered
}

func main() {
  flag.Usage = usage
  flag.Parse()
  if (*coverprof) == "" {
    flag.Usage()
    os.Exit(1)
  }

  encoder := xml.NewEncoder(os.Stdout)
  encoder.Indent("", "\t")

  summary := &Coverage {
    Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
  }

  pkg := &Package{
    BranchRate: 1,
    Classes: []*Class{},
  }

  profiles,err := cover.ParseProfiles(*coverprof)
  if err != nil {
    panic(err)
  }

  // Loop through files
  for _, profile := range profiles {
    if (pkg.Name == "") {
      pkg.Name = PackageName(profile)
    }

    classes := make(map[string]*Class)
    total, covered := profileCoverage(profile)
    summary.LinesValid+=total
    summary.LinesCovered+=covered

    funcs, err := FindFuncs(profile.FileName)
    if err != nil {
      panic(err)
    }

    for _,f := range funcs {
      class, ok := classes[f.Receiver]
      if !ok {
        class = &Class{
          Name:     f.Receiver,
          Filename: profile.FileName,
          Lines:    []*Line{},
        }

        classes[f.Receiver] = class
        pkg.Classes = append(pkg.Classes, class)
      }

      method := &Method{
        Name:  f.Name,
        Lines: []*Line{},
      }

      for _, b := range profile.Blocks {
        if b.StartLine > f.EndLine || (b.StartLine == f.EndLine && b.StartCol >= f.EndCol) {
          break
        }

        if b.EndLine < f.StartLine || (b.EndLine == f.StartLine && b.EndCol <= f.StartCol) {
          continue
        }

        for i := 0; i < b.NumStmt; i++ {
          line := &Line{
            Number: i + b.StartLine,
            Hits:   int64(b.Count),
            Branch: false,
          }

          method.Lines = append(method.Lines, line)
          class.Lines = append(class.Lines, line)
        }
      }

      class.Methods = append(class.Methods, method)
    }
  }

  pkg.LineRate = float64(summary.LinesCovered) / float64(summary.LinesValid)
  for _,class := range pkg.Classes {
    covered, total := 0,0
    for _, line := range class.Lines {
      if line.Hits > 0 {
        covered++
      }
      total++
    }
    class.LineRate = float64(covered) / float64(total)

    for _, method := range class.Methods {
      covered, total := 0,0
      for _, line := range method.Lines {
        if line.Hits > 0 {
          covered++
        }
        total++
      }

      method.LineRate = float64(covered) / float64(total)
    }
  }

  summary.BranchRate = 1
  summary.Packages = []*Package{ pkg }
  summary.LineRate = float64(summary.LinesCovered) / float64(summary.LinesValid)

  fmt.Printf(xml.Header)
  fmt.Printf("<!DOCTYPE coverage SYSTEM \"http://cobertura.sourceforge.net/xml/coverage-04.dtd\">\n")
  err = encoder.Encode(summary)
  if err != nil {
    panic(err)
  }

  fmt.Println()
}

func PackageName(p *cover.Profile) string {
  return filepath.Base(filepath.Dir(p.FileName))
}