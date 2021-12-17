package main

import (
    "fmt"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/client"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/helper"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/monitor"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/site"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/task"
    "github.com/KubeOperator/FusionComputeGolangSDK/pkg/vm"
    "math"
    "math/rand"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"
)

const (
    // FusionCompute登录相关参数
    hostUrl = "https://172.31.234.91:7443"
    username = "LiveMigr"
    password = "test@321"
    // 虚拟机Description和名称相关
    elasticDes = "0"
    softyDes = "1"
    participatedVMPrefix = "RabbitMQ-"
    // 输出文件名
    outputVmRealtimeFile = "vm.csv"
    outputHostRealtimeFile = "host.csv"
    // 算法相关参数
    maxMigrationNumPerHost = 5 // V1版本贪心算法的参数
    maxParallelMigrationNumPerHost = 1 // V2版本贪心算法的参数
    maxVmNumPerHost = 10 // 每台物理主机的最大虚拟机数量
    // 请求间隔, 用于防止FusionCompute崩溃
    taskStatusCheckInterval = 3  * time.Second
    migrationRequestInterval = 100 * time.Microsecond
)

func initParticipant(Vms []vm.Vm) ([]string, []string, map[string]string, map[string]string) {
    // 参与迁移的虚拟机
    vmNames := []string{
        "RabbitMQ-0", "RabbitMQ-1", "RabbitMQ-2", "RabbitMQ-3", "RabbitMQ-4", "RabbitMQ-5", "RabbitMQ-6", "RabbitMQ-7",
        "RabbitMQ-8", "RabbitMQ-9", "RabbitMQ-10", "RabbitMQ-11", "RabbitMQ-12", "RabbitMQ-13", "RabbitMQ-14",
        "RabbitMQ-15", "RabbitMQ-16", "RabbitMQ-17", "RabbitMQ-18", "RabbitMQ-19", "RabbitMQ-20", "RabbitMQ-21",
        "RabbitMQ-22", "RabbitMQ-23", "RabbitMQ-24", "RabbitMQ-25", "RabbitMQ-26", "RabbitMQ-27", "RabbitMQ-28",
        "RabbitMQ-29",
    }
    // 参与迁移的物理机
    hostNames := []string{"CNA04", "CNA05", "CNA06", "CNA07"}
    // vmName2HostName表示虚拟机的名字到物理机的名字的映射
    vmName2Class := make(map[string]string)
    // vmName2Class表示虚拟机的名字到物理机的名字的映射
    vmName2HostName := make(map[string]string)

    for _, v := range Vms {
        if strings.Contains(v.Name, participatedVMPrefix) {
            if _, err := strconv.Atoi(string(v.Name[len(participatedVMPrefix)])); err == nil {
                vmName2Class[v.Name], vmName2HostName[v.Name] = v.Description, v.HostName
            }
        }
    }
    return vmNames, hostNames, vmName2Class, vmName2HostName
}

func heuristicV1(Vms []vm.Vm, vmgr vm.Manager, taskManager task.Manager, vmname2uri, hostname2urn map[string]string) {
    vmNames, hostNames, vmName2Class, vmName2HostName := initParticipant(Vms)

    // 获取G矩阵: -1 代表弹性, 1 代表柔性
    G := make([][]int, len(vmNames))
    for i := 0; i < len(vmNames); i++ {
        G[i] = make([]int, len(hostNames))
        for j := 0; j < len(hostNames); j++ {
            if vmName2HostName[vmNames[i]] == hostNames[j] {
                if vmName2Class[vmNames[i]] == elasticDes {
                    G[i][j] = -1
                } else {
                    G[i][j] = 1
                }
            }
        }
    }

    // 开始对每一台物理机处理mn, mn数组必须始终对每一台物理机有效
    en := make([]int, len(hostNames))
    sn := make([]int, len(hostNames))
    mn := make([]int, len(hostNames))
    posNum, negNum := 0, 0
    zeroIndexes := make([]int, 0)
    // 将只有弹性虚拟机或者只有柔性虚拟机的主机排除在外 finished[j] = true
    finished := make([]bool, len(hostNames))
    for j := 0; j < len(hostNames); j++ {
        for i := 0; i < len(vmNames); i++ {
            if G[i][j] == -1 {
                en[j]++
            } else if G[i][j] == 1 {
                sn[j]++
            }
        }
        if en[j] > sn[j] {
            mn[j] = sn[j]
            posNum++
        } else if en[j] < sn[j] {
            mn[j] = -en[j]
            negNum++
        } else {
            zeroIndexes = append(zeroIndexes, j)
        }
        if en[j] == 0 || sn[j] == 0 {
            finished[j] = true
        }
    }

    for j := 0; j < len(zeroIndexes); j++ {
        if posNum <= negNum {
            mn[zeroIndexes[j]] = sn[zeroIndexes[j]]
            if sn[zeroIndexes[j]] > 0 {
                posNum++
            }
        } else {
            mn[zeroIndexes[j]] = -en[zeroIndexes[j]]
            if en[zeroIndexes[j]] > 0 {
                negNum++
            }
        }
    }
    fmt.Println(mn)

    // 迭代迁移, 迁移过程中需要更新mn, en, sn和vm2h; 不再更新G矩阵
    var wg sync.WaitGroup
    var mutex sync.Mutex
    startFirstly := math.MaxInt // 最早开始时间
    finishFinally := math.MinInt // 最迟结束时间
    for {
        minAbsMN := math.MaxInt                  // mn绝对值的最小值
        minAbsMNIndex := -1                      // mn绝对值的最小的下标
        startFirstlyThisParallel := math.MaxInt  // 这一趟迁移的最早开始时间
        finishFinallyThisParallel := math.MinInt // 这一趟迁移的最迟结束时间
        participated := make([]int, len(hostNames))  // 标记参与迁移的物理主机

        for j := 0; j < len(hostNames); j++ {
            if finished[j] == false { // 只有一种类型的虚拟机的物理机不再迁出
                if int(math.Abs(float64(mn[j]))) < minAbsMN {
                    minAbsMN, minAbsMNIndex = int(math.Abs(float64(mn[j]))), j
                }
            }
        }
        if minAbsMNIndex == -1 {
            break // 迁移完成, 所有物理机都已经finished
        }
        // 将min(minAbsMN, maxMigrationNumPerHost)台虚拟机依次迁出, 在下面的For循环中的所有迁移为一个轮并发迁移请求
        for i := 0; i < int(math.Min(float64(minAbsMN), maxMigrationNumPerHost)); i++ {
            // 除了要迁出的物理机之外, 在只有另一种类型的虚拟机的物理机中寻找剩余资源最多的一台
            minUsed := math.MaxInt
            minUsedIndex := -1
            for k := 0 ;; k++ {
                for j := 0; j < len(hostNames); j++ {
                    if minAbsMNIndex != j {
                        if (mn[j] * mn[minAbsMNIndex] < 0 || mn[j] == 0 && en[j] == 0 && mn[minAbsMNIndex] > 0 ||
                            mn[j] == 0 && sn[j] == 0 && mn[minAbsMNIndex] < 0) && participated[j] <= k {
                            if en[j] + sn[j] < minUsed {
                                minUsed, minUsedIndex = en[j] + sn[j], j
                            }
                        }
                    }
                }
                if minUsedIndex != -1 {
                    break
                }
            }
            participated[minUsedIndex]++
            src := 0
            dst := minUsedIndex
            for k := 0; k < len(vmNames); k++ {
                if vmName2HostName[vmNames[k]] == hostNames[minAbsMNIndex] {
                    if vmName2Class[vmNames[k]] == elasticDes && mn[minAbsMNIndex] < 0 ||
                        vmName2Class[vmNames[k]] == softyDes && mn[minAbsMNIndex] > 0 {
                        src = k
                        break
                    }
                }
            }
            rsp, err := vmgr.MigrateVM(vmname2uri[vmNames[src]], hostname2urn[hostNames[dst]], true)
            vmName2HostName[vmNames[src]] = hostNames[minUsedIndex]
            fmt.Print(hostNames[minAbsMNIndex] + "[" + vmNames[src] + "]" + " -> " + hostNames[dst] + ": ")
            if err != nil {
                fmt.Println("出现致命错误，算法停止!")
                helper.CheckError(err)
            }
            fmt.Println(*rsp)
            // 只更新源主机和目的主机的mn, en, sn; 不再更新G
            if mn[minAbsMNIndex] < 0 {
                en[minAbsMNIndex]--
                en[minUsedIndex]++
                mn[minAbsMNIndex]++
            } else {
                sn[minAbsMNIndex]--
                sn[minUsedIndex]++
                mn[minAbsMNIndex]--
            }
            if mn[minAbsMNIndex] == 0 {
                finished[minAbsMNIndex] = true
            }
            fmt.Println(mn)
            wg.Add(1)
            go func(src, dst int, rsp *vm.MigrateVmResponse) {
                defer wg.Done()
                for {
                    if trsp, _ := taskManager.Get(rsp.TaskUri); trsp != nil && trsp.Status == "success" {
                        start, err := strconv.Atoi(trsp.StartTime)
                        helper.CheckError(err)
                        finish, err := strconv.Atoi(trsp.FinishTime)
                        helper.CheckError(err)
                        mutex.Lock()
                        if startFirstly > start {
                            startFirstly = start
                        }
                        if finishFinally < finish{
                            finishFinally = finish
                        }
                        if startFirstlyThisParallel > start {
                            startFirstlyThisParallel = start
                        }
                        if finishFinallyThisParallel < finish {
                            finishFinallyThisParallel = finish
                        }
                        mutex.Unlock()
                        fmt.Printf(hostNames[minAbsMNIndex] + "[" + vmNames[src] + "]" +
                            " -> " + hostNames[dst] + " consumed %d sec!\n" ,
                            (finish - start) / 1000)
                        break
                    }
                    time.Sleep(taskStatusCheckInterval)
                }
            }(src, dst, rsp)
            time.Sleep(migrationRequestInterval)
        }  // 一个迁移序列, 一次请求结束
        wg.Wait()
        fmt.Println("已完成一趟并发迁移过程")
        fmt.Printf("这一趟并发迁移过程消耗了: %d秒!\n",
            (finishFinallyThisParallel - startFirstlyThisParallel) / 1000)
    } // 一趟并发迁移结束
    fmt.Printf("整个迁移规划过程消耗了: %d秒!\n" ,(finishFinally-startFirstly) / 1000)
}

func heuristicV2(Vms []vm.Vm, vmgr vm.Manager,  taskManager task.Manager, monitorManager monitor.Manager,
    vmname2uri, hostname2urn map[string]string) {
    vmNames, hostNames, vmName2Class, vmName2HostName := initParticipant(Vms)

    // 获取G矩阵: -1 代表弹性, 1 代表柔性
    G := make([][]int, len(vmNames))
    for i := 0; i < len(vmNames); i++ {
        G[i] = make([]int, len(hostNames))
    }
    for i := 0; i < len(vmNames); i++ {
        for j := 0; j < len(hostNames); j++ {
            if vmName2HostName[vmNames[i]] == hostNames[j] {
                if vmName2Class[vmNames[i]] == elasticDes {
                    G[i][j] = -1
                } else {
                    G[i][j] = 1
                }
            }
        }
    }

    // 开始对每一台物理机处理mn, mn数组必须始终对每一台物理机有效
    // 将只有弹性虚拟机或者只有柔性虚拟机的主机排除在外 finished[j] = true
    // 修改在弹性和柔性相等的情况下mn的取法，使得正负号的数量尽可能平衡
    en := make([]int, len(hostNames))
    sn := make([]int, len(hostNames))
    mn := make([]int, len(hostNames))
    posNum, negNum := 0, 0
    zeroIndexes := make([]int, 0)
    finished := make([]bool, len(hostNames))
    for j := 0; j < len(hostNames); j++ {
        for i := 0; i < len(vmNames); i++ {
            if G[i][j] == -1 {
                en[j]++
            } else if G[i][j] == 1 {
                sn[j]++
            }
        }
        if en[j] > sn[j] {
            mn[j] = sn[j]
            posNum++
        } else if en[j] < sn[j] {
            mn[j] = -en[j]
            negNum++
        } else {
            zeroIndexes = append(zeroIndexes, j)
        }
        if en[j] == 0 || sn[j] == 0 {
            finished[j] = true
        }
    }

    for j := 0; j < len(zeroIndexes); j++ {
        if posNum <= negNum {
            mn[zeroIndexes[j]] = sn[zeroIndexes[j]]
            if sn[zeroIndexes[j]] > 0 {
                posNum++
            }
        } else {
            mn[zeroIndexes[j]] = -en[zeroIndexes[j]]
            if en[zeroIndexes[j]] > 0 {
                negNum++
            }
        }
    }
    fmt.Println(mn)

    // 迭代迁移过程中需要更新mn, en, sn和vm2h; 不再更新G矩阵
    startFirstly := math.MaxInt // 最早开始时间
    finishFinally := math.MinInt // 最迟结束时间
    isFinish := false
    for ; isFinish == false; {
        // 每次循环表示一趟并发迁移
        var wg sync.WaitGroup
        var mutex sync.Mutex
        participated := make([]int, len(hostNames)) // 标记参与迁移的物理主机
        startFirstlyThisParallel := math.MaxInt     // 这一趟迁移的最早开始时间
        finishFinallyThisParallel := math.MinInt    // 这一趟迁移的最迟结束时间
        requestNumSent := 0

        for { // 每次循环表示一趟并发迁移内的一个请求
            src, dst := -1, -1  // 源虚拟机和目的物理主机的下标
            leastVM := math.MaxInt // 可作为目的物理主机中具有的最少虚拟机数量
            indexOfHostOfLeastVM := -1 // 对应具有的最少虚拟机数量的物理主机的下标
            minAbsMN := math.MaxInt // mn绝对值的最小值
            minAbsMNIndex := -1     // mn绝对值的最小的下标

            for i := 0; i < maxParallelMigrationNumPerHost && minAbsMNIndex == -1; i++ { // 按第一优先级寻找源主机
                for j := 0; j < len(hostNames); j++ {
                    if finished[j] == false && participated[j] <= i { // 参与过迁移的物理主机也不在参与迁出
                        if int(math.Abs(float64(mn[j]))) < minAbsMN {
                            minAbsMN, minAbsMNIndex = int(math.Abs(float64(mn[j]))), j
                        }
                    }
                }
            }
            if minAbsMNIndex == -1 {
                break
            }
            for i := 0; i < maxParallelMigrationNumPerHost && indexOfHostOfLeastVM == -1; i++ {
                for j := 0; j < len(hostNames); j++ {
                    if minAbsMNIndex != j && participated[j] <= i {
                        if mn[j] * mn[minAbsMNIndex] < 0 || mn[j] == 0 && en[j] == 0 && mn[minAbsMNIndex] > 0 ||
                            mn[j] == 0 && sn[j] == 0 && mn[minAbsMNIndex] < 0 {
                            if en[j]+sn[j] < leastVM {// 在符合要求的目的主机中寻找CPU利用率最低的
                                leastVM, indexOfHostOfLeastVM = en[j]+sn[j], j
                            }
                        }
                    } // 只考虑没有参与迁移的主机作目的主机
                }
            }
            if indexOfHostOfLeastVM == -1 {
                break // 找不到合适的目的主机, 该趟迁移结束
            }

            dst = indexOfHostOfLeastVM
            for k := 0; k < len(vmNames); k++ { // 在源主机中寻找一台类型符合的虚拟机迁出
                if vmName2HostName[vmNames[k]] == hostNames[minAbsMNIndex] {
                    if vmName2Class[vmNames[k]] == elasticDes && mn[minAbsMNIndex] < 0 ||
                        vmName2Class[vmNames[k]] == softyDes && mn[minAbsMNIndex] > 0 {
                        src = k
                        break
                    }
                }
            }

            // 将决定好的源和目的物理主机打上标记，表示参与了该趟迁移
            participated[minAbsMNIndex]++
            participated[dst]++
            // 只更新源主机和目的主机的mn, en, sn; 不再更新G
            if mn[minAbsMNIndex] < 0 {
                en[minAbsMNIndex]--
                en[indexOfHostOfLeastVM]++
                mn[minAbsMNIndex]++
            } else {
                sn[minAbsMNIndex]--
                sn[indexOfHostOfLeastVM]++
                mn[minAbsMNIndex]--
            }
            // 更新finished数组, 看是否有新的单类虚拟机物理主机
            if mn[minAbsMNIndex] == 0 {
                finished[minAbsMNIndex] = true
            }

            // 发送这一个发送请求
            rsp, err := vmgr.MigrateVM(vmname2uri[vmNames[src]], hostname2urn[hostNames[dst]], true)
            vmName2HostName[vmNames[src]] = hostNames[indexOfHostOfLeastVM]
            fmt.Print(hostNames[minAbsMNIndex] + "[" + vmNames[src] + "]" + " -> " + hostNames[dst] + ": ")
            if err != nil {
                fmt.Println("Fatal error occurs, algorithm stopped!")
                helper.CheckError(err)
            }
            fmt.Println(*rsp)
            fmt.Println(mn)
            // 创建协程轮询返回迁移结果
            wg.Add(1)
            requestNumSent++
            go func(src, dst int, rsp *vm.MigrateVmResponse) {
                for {
                    trsp, _ := taskManager.Get(rsp.TaskUri)
                    if trsp != nil && trsp.Status == "success" {
                        start, _ := strconv.Atoi(trsp.StartTime)
                        finish, _ := strconv.Atoi(trsp.FinishTime)
                        mutex.Lock()
                        if startFirstly > start {
                            startFirstly = start
                        }
                        if finishFinally < finish {
                            finishFinally = finish
                        }
                        if startFirstlyThisParallel > start {
                            startFirstlyThisParallel = start
                        }
                        if finishFinallyThisParallel < finish {
                            finishFinallyThisParallel = finish
                        }
                        mutex.Unlock()
                        fmt.Printf(hostNames[minAbsMNIndex] + "[" + vmNames[src] + "]" +
                            " -> "+hostNames[dst]+" consumed %d sec!\n",
                            (finish-start)/1000)
                        break
                    }
                    time.Sleep(taskStatusCheckInterval)
                }
                wg.Done()
            }(src, dst, rsp)
        } // 一个迁移序列, 一次请求结束
        wg.Wait()
        fmt.Println("已完成一趟并发迁移过程")
        fmt.Printf("这一趟并发迁移过程消耗了: %d秒!\n",
            (finishFinallyThisParallel - startFirstlyThisParallel) / 1000)
        isFinish = true
        for i := 0; i < len(hostNames); i++ {
            if mn[i] != 0 {
                isFinish = false
            }
        }
    } // 一趟并发迁移结束
    fmt.Printf("整个迁移规划过程消耗了: %d秒!\n" ,(finishFinally - startFirstly) / 1000)
    return
}

func shuffle(Vms []vm.Vm, vmgr vm.Manager, taskManager task.Manager, vmname2uri, hostname2urn map[string]string) {
    // 参与迁移的虚拟机和物理机
    vms, hosts, _, vm2h := initParticipant(Vms)

    var wg sync.WaitGroup
    startAll := math.MaxInt
    finishAll := math.MinInt

    hostVmNum := make([]int, len(hosts))
    for _, v := range vms {
    reselect:
        rand.Seed(time.Now().UnixNano())
        dst := rand.Intn(len(hosts))
        if hostVmNum[dst] >= maxVmNumPerHost {
            goto reselect
        }
        hostVmNum[dst]++
        if vm2h[v] == hosts[dst] {
            continue
        }
        rsp, err := vmgr.MigrateVM(vmname2uri[v], hostname2urn[hosts[dst]], true)
        wg.Add(1)
        fmt.Print(v + " -> " + hosts[dst] + ": ")
        if err != nil {
            fmt.Println("Fatal error occurs, shuffle stopped!")
            helper.CheckError(err)
        }
        fmt.Println(*rsp)
        go func(v string, dst int, rsp *vm.MigrateVmResponse) {
            defer wg.Done()
            for {
                trsp, err := taskManager.Get(rsp.TaskUri)
                helper.CheckError(err)
                if trsp != nil && trsp.Status == "success" {
                    start, err := strconv.Atoi(trsp.StartTime)
                    helper.CheckError(err)
                    finish, err := strconv.Atoi(trsp.FinishTime)
                    helper.CheckError(err)
                    if startAll > start {
                        startAll = start
                    }
                    if finishAll < finish{
                        finishAll = finish
                    }
                    fmt.Printf(v + " -> " + hosts[dst] + " consumed %d sec!\n" , (finish - start) / 1000)
                    break
                }
                time.Sleep(taskStatusCheckInterval)
            }
        }(v, dst, rsp)
        time.Sleep(migrationRequestInterval)
    }
    wg.Wait()
    fmt.Printf("TIME CONSUMED FOR THE WHOLE SUFFLES: %d sec!\n" ,(finishAll - startAll) / 1000)
}

func getPlacement(Vms []vm.Vm) [][]int {
    vms, hosts, vm2class, vm2itsHost := initParticipant(Vms)
    // 获取G矩阵: -1 代表弹性, 1 代表柔性
    G := make([][]int, len(vms))
    for i := 0; i < len(vms); i++ {
        G[i] = make([]int, len(hosts))
    }
    for i := 0; i < len(vms); i++ {
        for j := 0; j < len(hosts); j++ {
            if vm2itsHost[vms[i]] == hosts[j] {
                if vm2class[vms[i]] == elasticDes {
                    G[i][j] = -1
                } else {
                    G[i][j] = 1
                }
            }
        }
    }
    fmt.Println("{")
    for _, g := range G {
        fmt.Print("{")
        for j, gg := range g {
            fmt.Printf("%2d", gg)
            if j != len(g) - 1 {
                fmt.Print(",")
            }
        }
        fmt.Print("},")
        fmt.Println()
    }
    fmt.Println("}")
    return G
}

func setPlacement(Vms []vm.Vm, vmgr vm.Manager, taskManager task.Manager, vmname2uri, hostname2urn map[string]string,
    G [][]int) {
    vms, hosts, _, _ := initParticipant(Vms)
    OG := getPlacement(Vms) // 原来的关系矩阵

    var wg sync.WaitGroup
    for i := 0; i < len(G); i++ {
        for j := 0; j < len(G[0]); j++ {
            if OG[i][j] == 0 && G[i][j] != 0 { // 必须迁移到别的物理主机上面去
                rsp, err := vmgr.MigrateVM(vmname2uri[vms[i]], hostname2urn[hosts[j]], true)
                if err != nil {
                    fmt.Println("Fatal error occurs, shuffle stopped!")
                    helper.CheckError(err)
                }
                wg.Add(1)
                go func(v string, dst int, rsp *vm.MigrateVmResponse) {
                    defer wg.Done()
                    for {
                        trsp, err := taskManager.Get(rsp.TaskUri)
                        helper.CheckError(err)
                        if trsp != nil && trsp.Status == "success" {
                            start, err := strconv.Atoi(trsp.StartTime)
                            helper.CheckError(err)
                            finish, err := strconv.Atoi(trsp.FinishTime)
                            helper.CheckError(err)
                            fmt.Printf(v + " -> " + hosts[dst] + " consumed %d sec!\n" , (finish - start) / 1000)
                            break
                        }
                        time.Sleep(taskStatusCheckInterval)
                    }
                }(vms[i], j, rsp)
            }
        }
    }
    wg.Wait()
}

func debug(Vms []vm.Vm, vmgr vm.Manager, taskManager task.Manager, monitorManager monitor.Manager, vmname2uri,
    hostname2urn map[string]string) { // 用于调试
    // TODO: 编写代码用于调试
}

func main() {
    // 检查参数
    args := os.Args
    list := false
    if len(args) == 1 || args[1] == "--help" || args[1] == "-h" {
        helper.PrintUsageAndExit()
    } else if args[1] == "--list" || args[1] == "-l" {
        list = true
    }

    // 创建一个客户端并登并查询第一个站点初始化
    c := client.NewFusionComputeClient(hostUrl, username, password)
    helper.CheckError(c.Connect())
    s, vms, hosts := site.MetaCheckSite(c, list)
    vmManager := vm.NewManager(c, s.Uri)
    monitorManager := monitor.NewManager(c, s.Uri)
    taskManager := task.NewManager(c, s.Uri)

    vmMetrics := []string{helper.CpuUsage, helper.MemUsage, helper.DiskUsage, helper.DiskIOIn, helper.DiskIOOut,
        helper.NicByteIn, helper.NicByteOut}
    hostMetrics := []string{helper.DomUCpuUsage, helper.Dom0CpuUsage, helper.DomUMemUsage, helper.Dom0MemUsage,
        helper.LogicDiskUsage, helper.DiskIOIn, helper.DiskIOOut, helper.NicByteIn, helper.NicByteOut}
    helper.ChangeWorkDir2ExecDir() // 将当前工作目录更改为可执行文件所在的目录
    // 建立一些映射
    vmName2Uri := make(map[string]string, 0)
    hostName2Urn := make(map[string]string, 0)
    for _, v := range vms {
        vmName2Uri[v.Name] = v.Uri
    }
    for _, h := range hosts {
        hostName2Urn[h.Name] = h.Urn
    }
    if list == false { // 根据选项开始执行程序
        if args[1] == "--realtime" || args[1] == "-r" {
            // 创建并打开输出文件
            vmFile, err := os.Create(outputVmRealtimeFile)
            helper.CheckError(err)
            defer helper.CloseFileSafely(vmFile)
            hostFile, err := os.Create(outputHostRealtimeFile)
            helper.CheckError(err)
            defer helper.CloseFileSafely(hostFile)
            // 查询VMs的Realtime详细信息
            _, err = vmFile.WriteString("no,name,hostname")
            helper.CheckError(err)
            for _, m := range vmMetrics {
                _, err = vmFile.WriteString("," + m)
                helper.CheckError(err)
            }
            _, err = vmFile.WriteString("\n")
            helper.CheckError(err)
            for i, v := range vms {
                if !v.IsTemplate {
                    hostName, err := vmManager.GetHostNameOf(v.Uri)
                    helper.CheckError(err)
                    fmt.Printf("%02d  %-20s"+hostName, i, v.Name)
                    _, err = vmFile.WriteString(strconv.Itoa(i) + "," + v.Name + "," + hostName)
                    helper.CheckError(err)
                    rtd, err := monitorManager.GetObjectMetricRealtimeData(v.Urn, vmMetrics)
                    helper.CheckError(err)
                    for i := 0; i < len(vmMetrics); i++ {
                        value := rtd.Items[0].Value[i]
                        mValue, err := strconv.ParseFloat(value.MetricValue, 32)
                        helper.CheckError(err)
                        fmt.Printf("    "+value.MetricId+":"+"%9.2f", mValue)
                        fmt.Print(value.Unit)
                        _, err = vmFile.WriteString("," + value.MetricValue + value.Unit)
                        helper.CheckError(err)
                    }
                    fmt.Println()
                    _, err = vmFile.WriteString("\n")
                    helper.CheckError(err)
                }
            }
            fmt.Println()
            // 查询Hosts的Realtime详细信息
            _, err = hostFile.WriteString("no,name")
            helper.CheckError(err)
            for _, m := range hostMetrics {
                _, err = hostFile.WriteString("," + m)
                helper.CheckError(err)
            }
            _, err = hostFile.WriteString("\n")
            helper.CheckError(err)
            for i, h := range hosts {
                fmt.Printf("%02d"+"  "+h.Name, i)
                _, err = hostFile.WriteString(strconv.Itoa(i) + "," + h.Name)
                helper.CheckError(err)
                rtd, err := monitorManager.GetObjectMetricRealtimeData(h.Urn, hostMetrics)
                helper.CheckError(err)
                for i := 0; i < len(hostMetrics); i++ {
                    value := rtd.Items[0].Value[i]
                    mValue, err := strconv.ParseFloat(value.MetricValue, 32)
                    helper.CheckError(err)
                    fmt.Printf("    "+value.MetricId+":"+"%9.2f", mValue)
                    fmt.Print(value.Unit)
                    _, err = hostFile.WriteString("," + value.MetricValue + value.Unit)
                    helper.CheckError(err)
                }
                fmt.Println()
                _, err = hostFile.WriteString("\n")
                helper.CheckError(err)
            }
            fmt.Println()
        } else if args[1] == "--migrate" {
            if len(args) < 4 {
                fmt.Println("Parameters' number error!")
                goto exit
            } else {
                rsp, merr := vmManager.MigrateVM(vmName2Uri[args[2]], hostName2Urn[args[3]], true)
                helper.CheckError(merr)
                fmt.Println("Migration Request Sent!")
                fmt.Println(*rsp)
            }
        } else if args[1] == "--multigrate" || args[1] == "-m" {
            if len(args) < 4 {
                fmt.Println("Parameters' number error!")
                goto exit
            } else {
                src := 2
                dst := 3
                fil := 0
                tol := 0
                for dst < len(args) {
                    if _, merr := vmManager.MigrateVM(vmName2Uri[args[src]], hostName2Urn[args[dst]], true);
                    merr != nil {
                        fmt.Println("The request of migrating " + args[src] + " to " + args[dst] +
                            " sent unsuccessfully!")
                        fmt.Print("Reason: ")
                        fmt.Println(merr)
                        fil++
                    } else {
                        fmt.Println("The request of migrating " + args[src] + " to " + args[dst] +
                            " sent successfully!")
                    }
                    src += 2
                    dst += 2
                    tol++
                }
                fmt.Println("Request Success: " + strconv.Itoa(tol-fil))
                fmt.Println("Request Failure: " + strconv.Itoa(fil))
                fmt.Println("Total Request Sent:  " + strconv.Itoa(tol))
            }
        } else if args[1] == "--heuristic-v1" || args[1] == "-hv1" { // V1版本的贪心算法
            heuristicV1(vms, vmManager, taskManager, vmName2Uri, hostName2Urn)
        } else if args[1] == "--heuristic-v2" || args[1] == "-hv2" || args[1] == "--heuristic" { // V2版本的贪心算法
            heuristicV2(vms, vmManager, taskManager, monitorManager, vmName2Uri, hostName2Urn)
        } else if args[1] == "--shuffle" || args[1] == "-s" { // 随机打乱虚拟机放置
            shuffle(vms, vmManager, taskManager, vmName2Uri, hostName2Urn)
        } else if args[1] == "--debug" || args[1] == "-d" { // 用于调试的函数
            debug(vms, vmManager, taskManager, monitorManager, vmName2Uri, hostName2Urn)
        } else if args[1] == "--get-topology" || args[1] == "-gt" ||
            args[1] == "--get-placement" || args[1] == "-gp"  { // 获取当前的虚拟机放置
            _ = getPlacement(vms)
        } else if args[1] == "--set-topology" || args[1] == "-st" ||
            args[1] == "--set-placement" || args[1] == "-sp" {
            setPlacement(vms, vmManager, taskManager, vmName2Uri, hostName2Urn, [][]int{{ 0, 0, 0,-1},{ 0, 1, 0, 0},{ 0, 0, 0,-1},{ 0, 0, 1, 0},{ 0, 0,-1, 0},{ 0, 0, 1, 0},{ 0,-1, 0, 0},{ 1, 0, 0, 0},{ 0, 0, 0,-1},{ 0, 0, 1, 0},{ 0, 0, 0,-1},{ 0, 0, 0, 1},{-1, 0, 0, 0},{ 1, 0, 0, 0},{-1, 0, 0, 0},{ 1, 0, 0, 0},{-1, 0, 0, 0},{ 0, 0, 1, 0},{ 0, 0, 0,-1},{ 0, 0, 1, 0},{ 0, 0, 0,-1},{ 0, 1, 0, 0},{ 0, 0, 0,-1},{ 0, 0, 0, 1},{-1, 0, 0, 0},{ 0, 1, 0, 0},{ 0, 0, 0,-1},{ 1, 0, 0, 0},{ 0,-1, 0, 0},{ 0, 0, 1, 0}})
        }
        // 客户端登出
    }

exit:
    _ = c.DisConnect()
}

