package main
import (
  "fmt"
  "strings"
  "visor/internal/memory"
)
func main(){
 s,_:=memory.NewStore("data/memories"); all,_:=s.ReadAll(); terms:=[]string{"elevenlabs","excited","thoughtful","tags","vorgelesen"}; c:=0
 for i:=len(all)-1;i>=0;i--{t:=strings.ToLower(all[i].Text); ok:=false; for _,term:= range terms { if strings.Contains(t,term){ok=true;break} }; if ok {fmt.Println(all[i].Text); c++; if c>=6 {break}} }
}
