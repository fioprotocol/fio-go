# fiotop

A simple utility for watching the state of a node. Shows some basic net
info, peer info (if the net_api_plugin is enabled), database size (if
the db_size_api_plugin is enabled), list of producers, and an event stream.

Default is to connect to `http://localhost:8888` to specify a different
nodeos server, run with the `-u` option.

![Screenshot of fiotop running](fiotop.gif)

```
Keys:
    ? or F1 for help screen
    q or CTRL-C to exit
    r or CTRL-L to repaint screen
    d or CTRL-U to clear data
```

Note: some care has been taken to ensure that the display doesn't get strange
artifacts, but sometimes it happens. One thing that seems to help is using a
font that handles double-wide characters correctly such as those at
https://www.nerdfonts.com/. But, if despite this it still happens, just press
"r" to forcefully repaint the grid.
