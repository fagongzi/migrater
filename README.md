# migrater
gateway元数据迁移工具，用来迁移老版本的gateway的元数据到新版gateway上

## 编译
```bash
cd $GOPATH/src/github.com/fagongzi
git clone https://github.com/fagongzi/migrater.git
cd migrater
go build ./...
```

## 运行
migrater包含2个命令行参数：

* addr-api
  
  新版gateway的api server的GRPC服务地址

* cfg

  老gateway的proxy启动配置文件
