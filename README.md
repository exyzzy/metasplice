# metasplice

Automatically splice diffs into a metaapi project, requires metaproj and metaapi

The current version matches Medium article:
* Automatic Applications in Go (in progress)


See also:
* http://github.com/exyzzy/metaproj
* http://github.com/exyzzy/metaapi

## Most Common Scenario:

1. Create a metaapi project
```
#assume project and database name: todo (can be anything)
#assume sql file: events.sql (from examples, but can be anything)
createuser -P -d todo <pass: todo>
createdb todo
go get github.com/exyzzy/metaapi
go install $GOPATH/src/github.com/exyzzy/metaapi
go get github.com/exyzzy/metaproj
go install $GOPATH/src/github.com/exyzzy/metaproj
go get github.com/exyzzy/metasplice
go install $GOPATH/src/github.com/exyzzy/metasplice
cp $GOPATH/src/github.com/exyzzy/metaapi/examples/events.sql .
metaproj -sql=events.sql -proj=todo -type=vue
cd todo
go generate
go install
go test
```
2. Create the extractsplice folder
```
mkdir todo/extractsplice
#create todo/extractsplice/extractsplice.go (see below)
```

3. Edit within splice points, or add new files
4. Update extractsplice.go to track changes
5. Extract the diffs
```
cd todo/extractsplice
go generate
```
6. Change .sql file, delete or rename the existing metaapi project, make new metaapi project
7. Apply the extracted diffs to the new project
```
cd tododiff/applysplice
```


## extractsplice/extractsplice.go template:
```
package splice
//go:generate  mkdir -p ../../tododiff
//go:generate  mkdir -p ../../tododiff/templates
//go:generate  mkdir -p ../../tododiff/extractsplice
//go:generate  cp ../extractsplice/extractsplice.go ../../tododiff/extractsplice/extractsplice.go
//place any new mkdir -p ../../tododiff/<somefolder> commands here
//place any new cp ../<somefolder>/<somefile> ../../tododiff/<somefolder>/<somefile> commands here
//place any metasplice -src=../<somefolder>/<somefile> -dest=../../tododiff/<somefolder>/<somedifffile> -mode=extract
//go:generate  metasplice -src=../extractsplice/extractsplice.go -dest=../../tododiff/applysplice/applysplice.go -mode=applyfile
//go:generate cp -R ../.git ../../tododiff/.git
//go:generate  cp ../todos.sql ../../tododiff/applysplice/todos.sql
```