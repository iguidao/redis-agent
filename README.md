# redis-agent
redis的客户端，进行一些信息收集操作

### 功能列表
1. 检测redis是否有进行save操作，并上传到腾讯COS里

### 配置更改
- 打开main.go文件更改下面四个配置
1. AccessKey
2. AccessKeyID
3. EndpointPub
4. RedisManagerUrl
### 启动方式
- 执行 go mod 初始化
- 执行 CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o redis-agent main.go 打包
- 放入系统crontab中，执行redis-agent

### 联系方式
暂无


