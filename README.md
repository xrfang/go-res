# Go Resource Packager

`go-res` is a minimalistic resource packager for creating battery-included Go applications.  It is very simple comparing to the popular [go-bindata](https://github.com/go-bindata/go-bindata) utility: `go-bindata` has nearly 100 funcs scattered in dozens of Go source file while `go-res` has only one source file with 5 funcs, of which only 2 are exported!

## The Idea

`go-res` provides a simple way to append all files in a specified directory (and all its sub-directories) at the end of the Go executable as tar.gz data, which in turn can be extracted on-demand.  In another word, it is a backpack where the content must be taken out when use.

## The APIs

### Pack

    func Pack(root string) error

Pack collect all files under directory `root` and its sub-directories, append them as tar.gz data at the end of the running application, then add a signature at the end to make the Pack action idempotent -- you can pack any directory many times, only the last operation's result is kept, all previous ones are discarded.

### Extract

    func Extract(path string, policy ExtractPolicy) error

Extract extracts embeded resources to the `path` specified. `policy` is used to control content overwriting behavior:

|policy  |logic  |typical use  |
|-- |-- |--|
|**NoOverwrite**|if a file exists at destination location, it will _not_ be overwritten||
|**OverwriteIfNewer**|only overwrite a file if the one in resource pack is newer||
|**AlwaysOverwrite**|always overwrite file at destination||
|**Verbatim**|remove `path` tree if it exists, then extract resource to `path`, creating all directories as needed||

## The Use Case

## The Pros & Cons