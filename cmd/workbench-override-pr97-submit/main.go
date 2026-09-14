package main

import (
 "encoding/json"
 "fmt"
 "os"
 "github.com/DaisyCloverSoftware/workbench/internal/core"
)
func main(){
 if len(os.Args)!=2{fmt.Fprintln(os.Stderr,"one registered Windows host id is required");os.Exit(2)}
 job,err:=core.SubmitOverridePR97ProofJob(os.Args[1]);if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 if err:=json.NewEncoder(os.Stdout).Encode(job);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}
