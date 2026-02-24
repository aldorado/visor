package main
import (
  "fmt"
  "os"
  "visor/internal/memory"
)
func main(){
  key:=os.Getenv("OPENAI_API_KEY")
  if key=="" { fmt.Println("OPENAI_API_KEY missing"); return }
  m,err:=memory.NewManager("data", key)
  if err!=nil { panic(err) }
  out,err:=m.Lookup("welches model nutzt visor",5)
  if err!=nil { panic(err) }
  fmt.Println(out)
}
