package slice

import (
    "fmt"
)

//删除
func Remove(slice []interface{}, i int) []interface{} {
    //    copy(slice[i:], slice[i+1:])
    //    return slice[:len(slice)-1]
    return append(slice[:i], slice[i+1:]...)
}

//新增
func add(slice []interface{}, value interface{}) []interface{} {
    return append(slice, value)
}

//插入
func insert(slice *[]interface{}, index int, value interface{}) {
    rear := append([]interface{}{}, (*slice)[index:]...)
    *slice = append(append((*slice)[:index], value), rear...)
}

//修改
func update(slice []interface{}, index int, value interface{}) {
    slice[index] = value
}

//查找
func find(slice []interface{}, index int) interface{} {
    return slice[index]
}

//清空slice
func Empty(slice *[]interface{}) {
    //    *slice = nil
    *slice = append([]interface{}{})
}

//遍历
func list(slice []interface{}) {
    for _, v := range slice {
        fmt.Printf("%d ", v)
    }
}
 