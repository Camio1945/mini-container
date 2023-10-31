package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// go run main.go <cmd> <args>
// 注：os.Args[0]是可执行文件本身的路径，大概长这样：/tmp/go-build150488936/b001/exe/main
func main() {
	fmt.Printf("Running %v \n", os.Args[1:])
	createCgroupFiles() // 创建cgroup相关的文件夹和文件
	cmd := buildCmd()   // 构建命令
	prepareContainer()  // 准备容器
	defer unmount()     // 取消挂载，延迟进行
	must(cmd.Run())     // 执行命令，前面调用了syscall.Chroot，这里实际执行的是：/home/liz/ubuntufs/bin/bash
}

/** 取消挂载 */
func unmount() {
	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("mytemp", 0))
}

/** 准备容器 */
func prepareContainer() {
	must(syscall.Sethostname([]byte("container"))) // 设置容器的主机名
	must(syscall.Chroot("/home/liz/ubuntufs"))     // 修改当前进程的根目录，这里的参数理论上不应该写死
	must(os.Chdir("/"))                            // 修改当前进程的工作目录
	// 挂载proc文件系统，挂载点为/proc，容器需要使用proc文件系统来获取进程相关信息；第1个参数不重要
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	// 挂载tmpfs文件系统，挂载点为/mytemp，容器需要使用tmpfs文件系统来创建临时文件
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))
}

/** 构建命令 */
func buildCmd() *exec.Cmd {
	cmd := exec.Command(os.Args[1], os.Args[2:]...) // 示例：/bin/bash ，本例中 os.Args[2:] 为空
	cmd.Stdin = os.Stdin                            // 标准输入
	cmd.Stdout = os.Stdout                          // 标准输出
	cmd.Stderr = os.Stderr                          // 标准错误
	return cmd
}

/** 创建cgroup相关的文件夹和文件 */
func createCgroupFiles() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")         // 值：/sys/fs/cgroup/pids
	_ = os.Mkdir(filepath.Join(pids, "liz"), 0755) // 创建文件夹：/sys/fs/cgroup/pids/liz
	// 写文件：/sys/fs/cgroup/pids/liz/pids.max，限制最大进程数为20
	must(os.WriteFile(filepath.Join(pids, "liz/pids.max"), []byte("20"), 0700))
	// 写文件：/sys/fs/cgroup/pids/liz/notify_on_release，当容器退出时，自动清理cgroup
	must(os.WriteFile(filepath.Join(pids, "liz/notify_on_release"), []byte("1"), 0700))
	// 写文件：/sys/fs/cgroup/pids/liz/cgroup.procs，将当前进程的pid写入
	must(os.WriteFile(filepath.Join(pids, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

/** 要求操作必须成功，否则抛出异常（本例中会导致进程终止） */
func must(err error) {
	if err != nil {
		panic(err)
	}
}
