#!/bin/bash
# Name    : start_script.py
# Date    : 2016.03.28
# Func    : 启动脚本
# Note    : 注意：当前路径为应用部署文件夹

#############################################################
# 初始化环境

# 用户自定义
app_folder="kube-state-metrics-exporter"                 # 项目根目录
process_name="kube-state-metrics-exporter"       # 进程名

install_base="/usr/local/easyops"          # 安装根目录
data_base="/data/easyops"             # 日志/数据根目录

#############################################################
# 通用前置
# ulimit 设定
ulimit -n 100000
export LD_LIBRARY_PATH=/usr/local/easyops/ens_client/sdk:${LD_LIBRARY_PATH}

# 执行准备
install_path="${install_base}/${app_folder}/"
if [[ ! -d ${install_path} ]]; then
    echo "${install_path} is not exist"
    exit 1
fi


# 启动命令
start_cmd="./bin/kube-state-metrics-exporter --port=port1 --telemetry-port=port2 --kubeconfig=/root/.kube/config --apiserver=http://ip:port >/dev/null 2>log/${app_folder}.log &"


# 日志目录
log_path="${data_base}/${app_folder}/log"
mkdir -p ${log_path}

# 数据目录
data_path="${data_base}/${app_folder}/data"
mkdir -p ${data_path}


#############################################################

# 启动程序
echo "start by cmd: ${start_cmd}"
cd ${install_path} && eval "${start_cmd}"
if [[ $? -ne 0 ]];then
    echo "start error, exit"
    exit 1
fi
#############################################################