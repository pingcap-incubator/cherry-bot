package checkIssue

import (
	"fmt"
	"testing"
)

func TestFilterImg(t *testing.T) {
	//str1 := "sfdf"
	str := "352452aaaa 中文 <img >sss<><><><> <img width=\"956\" alt=\"屏幕快照 2021-02-03 下午8 13 16\" src=\"https://user-images.githubusercontent.com/5906259/106745942-c7cc2000-665c-11eb-9689-6bc8d77ce982.png\"> <><><>"
	fmt.Println(str)
	fmt.Println(filterImg(str))
}

func TestFilterSquareBracket(t *testing.T) {
	//str := "[中文]2525[]"
	str := "## Error Report\\r\\n\\r\\n**This repository is ONLY used to solve issues related to DOCS.\\r\\nFor other issues (related to TiDB, PD, etc), please move to [other repositories](https://github.com/pingcap/).**\\r\\n\\r\\nPlease answer the following questions before submitting your issue. Thanks!\\r\\n\\r\\n1. What is the URL/path of the document related to this issue?\\r\\n\\r\\nhttps://docs.pingcap.com/zh/tidb/dev/privilege-management\\r\\n\\r\\n2. How would you like to improve it?\\r\\n\\r\\ndelete grand in the table\\r\\n\\r\\n![企业微信截图_e7ea7242-875c-443f-9661-39bec203c1ee](https://user-images.githubusercontent.com/53471087/106870333-f228e680-670b-11eb-9048-14c1d5a729e7.png)\\r\\n\\r\\n\\r\\n\\r\\n\\r\\n"
	fmt.Println(str)
	fmt.Println(filterSquareBracket(str))
}

func TestFilterBackQuote(t *testing.T) {
	str := "```ǎ傦眢否畬傮Ȕ炏芭裪```"
	fmt.Println(str)
	fmt.Println(filterBackQuote(str))
}

func TestAll(t *testing.T) {
	//str := "<img> <img 中\n> <\r\n> <img\n\r> [\n\r] [\n] [<>] () (\"\") ```afdaf\n sfsfs ```  ```adfaf中文\n``` ```中文\n\r```"
	//str := "Please answer these questions before submitting your issue. Thanks!\r\n\r\n1. What did you do?\r\nImport a tpcc 1400 warehouses parquet data from s3, with default ulimit.\r\n\r\n2. What did you expect to see?\r\nLightning warning about the low nofile limit would crash the import procedure.\r\n\r\n\r\n3. What did you see instead?\r\nNothing about it was mentioned until lightning fail due to exceed the open file limit.\r\n\r\n4. What version of BR and TiDB/TiKV/PD are you using?\r\n\r\nCluster: v5.0.0-nightly (built in 2021-3-17)\r\n\r\n5. Operation logs  \r\nBecause this log contains some internal information, see it at [google drive](https://drive.google.com/file/d/1O81I4zpuNb6__vQNdw1tM_FSgCMKYTTr/view?usp=sharing).\r\n\r\n6. Configuration of the cluster and the task\r\n   - `tidb-lightning.toml` for TiDB-Lightning if possible\r\n```toml\r\n[lightning]\r\n# 日志\r\nlevel = \"debug\"\r\nfile = \"tidb-lightning#3.log\"\r\npprof-port = 8289\r\n\r\n[tikv-importer]\r\n# 选择使用的 local 后端\r\nbackend = \"local\"\r\n# 设置排序的键值对的临时存放地址，目标路径需要是一个空目录\r\n\"sorted-kv-dir\" = \"/lightning-data/sorted-kv-dir\"\r\n\r\n[mydumper]\r\n# 源数据目录。\r\ndata-source-dir = \"s3://tools-uw2/tpcc/100g-tpcc1400-parquet?region=us-west-2\"\r\n\r\nfilter = ['*.*', '!mysql.*', '!sys.*', '!INFORMATION_SCHEMA.*', '!PERFORMANCE_SCHEMA.*', '!METRICS_SCHEMA.*', '!INSPECTION_SCHEMA.*']\r\n\r\n[tidb]\r\n# 目标集群的信息\r\nhost = \"redacted\"\r\nport = 4000\r\nuser = \"root\"\r\npassword = \"redacted\"\r\n# 表架构信息在从 TiDB 的“状态端口”获取。\r\nstatus-port = 10080\r\n# 集群 pd 的地址\r\npd-addr = \"172.31.17.254:2379\"\r\n```"
	str := "## Bug Report\\r\\n\\r\\nPlease answer these questions before submitting your issue. Thanks!\\r\\n\\r\\n### 1. Minimal reproduce step (Required)\\r\\n```mysql\\r\\nmysql> CREATE TABLE `t1`  (\\r\\n  `COL1` varchar(20) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,PRIMARY KEY (`COL1`(5)) USING BTREE\\r\\n) ENGINE = InnoDB CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;\\r\\nmysql> insert into t1 values(\\\"ý忑辦孈策炠槝衧魮與\\\");\\r\\nmysql> insert into t1 values(\\\"ǎ傦眢否畬傮Ȕ炏芭裪\\\");\\r\\n```\\r\\n<!-- a step by step guide for reproducing the bug. -->\\r\\n\\r\\n### 2. What did you expect to see? (Required)\\r\\n```mysql\\r\\nmysql> select * from t1 where col1 between 0xC78EE582A6E79CA2E590A6E795ACE582AEC894E7828FE88AADE8A3AA and 0xC3BDE5BF91E8BEA6E5AD88E7AD96E782A0E6A79DE8A1A7E9ADAEE88887;\\r\\n+---------------------+\\r\\n| COL1                |\\r\\n+---------------------+\\r\\n| ǎ傦眢否畬傮Ȕ炏芭裪  |\\r\\n| ý忑辦孈策炠槝衧魮與 |\\r\\n+---------------------+\\r\\n2 rows in set (0.07 sec)\\r\\n\\r\\n```\\r\\n### 3. What did you see instead (Required)\\r\\n```mysql\\r\\nmysql> select * from t1 where col1 between 0xC78EE582A6E79CA2E590A6E795ACE582AEC894E7828FE88AADE8A3AA and 0xC3BDE5BF91E8BEA6E5AD88E7AD96E782A0E6A79DE8A1A7E9ADAEE88887;\\r\\nEmpty set\\r\\n\\r\\n```\\r\\n### 4. What is your TiDB version? (Required)\\r\\n```mysql\\r\\nRelease Version: v4.0.0-beta.2-2390-gfd706ab76\\r\\nEdition: Community\\r\\nGit Commit Hash: fd706ab76bd09ac859aa0a4de7fe9e07da3c5508\\r\\nGit Branch: master\\r\\nUTC Build Time: 2021-03-17 11:37:12\\r\\nGoVersion: go1.13\\r\\nRace Enabled: false\\r\\nTiKV Min Version: v3.0.0-60965b006877ca7234adaced7890d7b029ed1306\\r\\nCheck Table Before Drop: false\\r\\n```\\r\\n<!-- Paste the output of SELECT tidb_version() -->\\r\\n\\r\\n\""
	fmt.Println(str)
	fmt.Println("----")
	fmt.Println(filterImg(filterSquareBracket(filterBackQuote(str))))
}
