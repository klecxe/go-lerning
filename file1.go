package main
import "fmt"
func main () {
    arr := [100]int {}
    var a int
    fmt.Scan(&a)
   for i := 0; i < a; i++{
        fmt.Scan(&arr[i])
               }
       for inx,elem := range arr{
           if inx % 2 == 0 && elem != 0 {
           fmt.Printf("%d ", elem)
           }
   }
}
