// Package hostwiring connects hostbus input ports to legacy package-level channels in hostnotify, run, and remote.
// New code should prefer injecting dependencies; these bindings exist so feature packages can still emit host events
// without yet taking a HostPorts interface through every call site.
package hostwiring
