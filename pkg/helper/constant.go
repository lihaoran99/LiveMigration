package helper

const (
	//CpuUsage CPU占用率
	CpuUsage  = "cpu_usage"
	//MemUsage 内存利用率
	MemUsage  = "mem_usage"
	//DiskUsage 磁盘利用率
	DiskUsage = "disk_usage"

	NicInfo         	= "nic_info"
	NicOriginalInfo 	= "nic_original_info"
	NicDynamicInfo      = "nic_dynamic_info"

	LogicDisk           = "logic_disk"
	LogicDiskUsage		= "logic_disk_usage"

	DiskIOInfo = "disk_io_info"
	//DiskIOIn 磁盘IO写入
	DiskIOIn  = "disk_io_in"
	//DiskIOOut 磁盘IO读出
	DiskIOOut = "disk_io_out"

	//DomUCpuUsage 主机虚拟化域CPU占用率
	DomUCpuUsage     = "domU_cpu_usage"
	//DomUMemUsage 主机虚拟化域内存占用率
	DomUMemUsage     = "domU_mem_usage"
	//Dom0CpuUsage 主机管理域CPU占用率
	Dom0CpuUsage     = "dom0_cpu_usage"
	//Dom0MemUsage 主机管理域内存占用率
	Dom0MemUsage     = "dom0_mem_usage"
	Dom0StorageUsage = "dom0_storage_usage"

	NicNum     = "nic_num"
	NicDownNum = "nic_down_num"
	NicByteIn  = "nic_byte_in"
	NicByteOut = "nic_byte_out"

	HugepageResource    = "hugepageResource"

	DSReadPerSecond 	= "ds_read_per_second"	// 数据存储每秒读数据次数
	DSWritePerSecond	= "ds_write_per_second" // 数据存储每秒写数据次数
	DSSvctm				= "ds_svctm"			// 数据存储平均I/O处理时间
	DSAwite				= "ds_await"			// 数据存储I/O响应时延
)