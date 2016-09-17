# phrack-reader
A simple Go powered CLI reader for [Phrack Zine](http://phrack.org/).

Powered by [GoCui](https://github.com/jroimartin/gocui)!

# Installation
go install github.com/0x4445565a/phrack-reader

# Usage
```
phrack-reader
# or (Where 20 is the issue you want to read)
phrack-reader 20
```

# UI
The UI is simple.  It has three sections, the pager, body, and status.
To toggle between pager and body you use tab.
To move it is just arrow keys.
To load a different issue highlight "load" in the pager and type in your issue number (then hit enter).  It's all dead simple.

![Reading the most recent issue](http://i.imgur.com/i6GCwzH.png)
