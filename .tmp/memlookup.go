package main

import (
  "fmt"
  "strings"
  "visor/internal/memory"
)

func main() {
  s, err := memory.NewStore("data/memories")
  if err != nil { panic(err) }
  all, err := s.ReadAll()
  if err != nil { panic(err) }
  q := []string{"sprachnachricht", "elevenlabs", "visor", "diggi", "bro"}
  n := 0
  for i := len(all)-1; i >=0; i-- {
    t := strings.ToLower(all[i].Text)
    ok := false
    for _, term := range q {
      if strings.Contains(t, term) { ok = true; break }
    }
    if ok {
      fmt.Println("-", all[i].Text)
      n++
      if n >= 8 { break }
    }
  }
}
