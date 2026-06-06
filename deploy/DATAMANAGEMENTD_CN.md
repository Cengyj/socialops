# datamanagementd 部署边界说明

当前开源仓库不随附 datamanagementd 源码模块，也不随附可直接安装的 datamanagementd systemd 单元或安装脚本。

不要从当前仓库执行 datamanagementd 源码构建；当前仓库中的主服务只保留数据管理 agent 的兼容探测与接口边界。若后续需要启用外部数据管理 agent，应由提供该 agent 的发布物或单独模块给出对应部署文件。

当前主服务固定探测路径仍为 `/tmp/socialops-datamanagement.sock`。在没有可用 agent 发布物时，管理后台数据管理相关能力应保持未启用或不可连接状态。
