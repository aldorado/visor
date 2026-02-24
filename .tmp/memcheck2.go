package main
import (
  "fmt"
  "os"
  "visor/internal/memory"
)
func main(){
  key:=os.Getenv("OPENAI_API_KEY")
  if key=="" { fmt.Println("skip memory lookup: OPENAI_API_KEY missing"); return }
  m,err:=memory.NewManager("data",key); if err!=nil { fmt.Println(err); return }
  out,err:=m.Lookup("pi session slow no-session",3); if err!=nil { fmt.Println(err); return }
  if out=="" { fmt.Println("no memories") } else { fmt.Println(out) }
}
