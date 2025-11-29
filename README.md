# Movies API

<!-- 在这里写你的设计文档 -->
为什么选用mysql？
个人技术栈有限，mysql是强项

表设计
movies 表：存电影基本信息
movie_box_office 表：单独存票房数据，因为票房是从外部 API 拿的，分开存方便管理
movie_ratings 表：用 (movie_id, rater_id) 做联合主键，天然支持 upsert——同一个用户对同一部电影重复评分会自动覆盖

后端Go
技术栈要求使用golang
优点：天然支持高并发，编译成的话是单个二进制文件
缺点：开发效率相对较低，生态不如动态语言丰富
标准库的net/http就够用了，不需要引入很重的框架
处理并发简单，虽然这个项目用不太到，但goroutine确实方便

容器化
dockerfile
第一阶段用 golang 镜像编译
第二阶段只复制编译好的二进制文件到 alpine 镜像
最终镜像很小，而且用非 root 用户运行

dockercompose
mysql：数据库，带健康检查
migrations：用goose跑数据库迁移，跑完自动退出
api：应用本身，等migrations跑完才启动

待优化的点：
加 Redis 缓存热门电影数据，减轻数据库压力
评分聚合可以缓存，不用每次都算
票房API调用可以改成异步的，不阻塞创建接口