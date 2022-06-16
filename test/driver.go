package main

import (
  "os"
  "fmt"
  pa "path"
  "github.com/dianpeng/mono-service/pl"
)

type testResult struct {
  total int
  pass int
  compileFail int
  sessionFail int
  execFail int
}

func runAllTestFile(path string) (testResult, error) {
  fs, err := os.ReadDir(path)
  if err != nil {
    return testResult{}, err
  }

  t := testResult{}

  for _, f := range fs {
    if f.IsDir() {
      continue
    }

    fpath := pa.Join(path, f.Name())

    fmt.Printf(">> file: %s\n", fpath)

    data, err := os.ReadFile(fpath)
    if err != nil {
      return testResult{}, err
    }
    t.total++

    p, err := pl.CompilePolicy(string(data))
    if err != nil {
      t.compileFail++
    } else {
      ev := pl.NewEvaluatorSimple()
      if err := ev.EvalSession(p); err != nil {
        t.sessionFail++
      } else {
        if err := ev.Eval("test", p); err != nil {
          t.execFail++
        } else {
          t.pass++
        }
      }
    }
  }

  return t, nil
}

func main() {
  r, e := runAllTestFile("assets/test/")
  if e != nil {
    fmt.Printf("===============%s========\n", e.Error())
    return
  }

  fmt.Printf("================================\n")
  fmt.Printf("Total> %d\n", r.total)
  fmt.Printf("Pass> %d\n", r.pass)
  fmt.Printf("CompileFail> %d\n", r.compileFail)
  fmt.Printf("SessionFail> %d\n", r.sessionFail)
  fmt.Printf("ExecFail> %d\n", r.execFail)
  fmt.Printf("================================\n")
}
