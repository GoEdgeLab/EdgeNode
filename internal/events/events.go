package events

type Event = string

const (
	EventStart             Event = "start"             // start loading
	EventLoaded            Event = "loaded"            // first load
	EventQuit              Event = "quit"              // quit node gracefully
	EventReload            Event = "reload"            // reload config
	EventTerminated        Event = "terminated"        // process terminated
	EventNFTablesReady     Event = "nftablesReady"     // nftables ready
	EventReloadSomeServers Event = "reloadSomeServers" // reload some servers
)
