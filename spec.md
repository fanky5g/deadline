Package Deadline will present an interface (Contract) that specifies these behaviours:
- CheckTimeout: bool
- ExecuteOnTimeout:
  @param ...interface{}
  @return void

type Engine, will run as a background function(go ....) that continually loops over items in pool
Engine enters a sleep mode when theres no items in its deadline channel and wakes up when one or more objects are in pool
Engine will persist to localfile provided in the new function; if no file is presented, persistence mode will be disabled;