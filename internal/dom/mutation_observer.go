package dom

// MutationRecord represents a single DOM mutation.
type MutationRecord struct {
	Type            string // "childList", "attributes", "characterData"
	Target          Node
	AddedNodes      []Node
	RemovedNodes    []Node
	PreviousSibling Node
	NextSibling     Node
	AttributeName   string
	OldValue        string
}

// MutationObserverInit contains options for MutationObserver.Observe.
type MutationObserverInit struct {
	ChildList             bool
	Attributes            bool
	CharacterData         bool
	Subtree               bool
	AttributeOldValue     bool
	CharacterDataOldValue bool
	AttributeFilter       []string
}

// MutationCallback is the function invoked with queued mutation records.
type MutationCallback func(records []*MutationRecord, observer *MutationObserver)

// MutationObserver watches for DOM mutations and queues records.
type MutationObserver struct {
	callback MutationCallback
	records  []*MutationRecord
}

type registeredObserver struct {
	observer *MutationObserver
	options  MutationObserverInit
}

// observerRegistry maps each observed node to its registered observers.
var observerRegistry = map[Node][]*registeredObserver{}

// pendingObservers tracks observers with queued records awaiting delivery.
var pendingObservers []*MutationObserver

// NewMutationObserver creates a MutationObserver with the given callback.
func NewMutationObserver(callback MutationCallback) *MutationObserver {
	return &MutationObserver{callback: callback}
}

// Observe registers this observer on target with the given options.
// If this observer is already observing target, the options are replaced.
func (mo *MutationObserver) Observe(target Node, options MutationObserverInit) {
	regs := observerRegistry[target]
	for _, reg := range regs {
		if reg.observer == mo {
			reg.options = options
			return
		}
	}
	observerRegistry[target] = append(regs, &registeredObserver{
		observer: mo,
		options:  options,
	})
}

// Disconnect removes all registrations for this observer and clears pending records.
func (mo *MutationObserver) Disconnect() {
	for node, regs := range observerRegistry {
		var kept []*registeredObserver
		for _, reg := range regs {
			if reg.observer != mo {
				kept = append(kept, reg)
			}
		}
		if len(kept) == 0 {
			delete(observerRegistry, node)
		} else {
			observerRegistry[node] = kept
		}
	}
	mo.records = nil
	removePending(mo)
}

// TakeRecords returns and clears all queued records without invoking the callback.
func (mo *MutationObserver) TakeRecords() []*MutationRecord {
	records := mo.records
	mo.records = nil
	removePending(mo)
	return records
}

// FlushMutationObservers delivers queued records to all pending observers.
func FlushMutationObservers() {
	for len(pendingObservers) > 0 {
		observers := pendingObservers
		pendingObservers = nil
		for _, mo := range observers {
			if len(mo.records) == 0 {
				continue
			}
			records := mo.records
			mo.records = nil
			mo.callback(records, mo)
		}
	}
}

func removePending(mo *MutationObserver) {
	for i, o := range pendingObservers {
		if o == mo {
			pendingObservers = append(pendingObservers[:i], pendingObservers[i+1:]...)
			return
		}
	}
}

func enqueuePending(mo *MutationObserver) {
	for _, o := range pendingObservers {
		if o == mo {
			return
		}
	}
	pendingObservers = append(pendingObservers, mo)
}

// queueChildListMutation queues a childList mutation record on matching observers.
func queueChildListMutation(target Node, added, removed []Node, prev, next Node) {
	if target == nil {
		return
	}
	record := &MutationRecord{
		Type:            "childList",
		Target:          target,
		AddedNodes:      added,
		RemovedNodes:    removed,
		PreviousSibling: prev,
		NextSibling:     next,
	}
	notifyObservers(target, record)
}

// queueAttributeMutation queues an attributes mutation record on matching observers.
func queueAttributeMutation(target Node, name, oldValue string) {
	record := &MutationRecord{
		Type:          "attributes",
		Target:        target,
		AttributeName: name,
		OldValue:      oldValue,
	}
	notifyObservers(target, record)
}

// queueCharacterDataMutation queues a characterData mutation record.
func queueCharacterDataMutation(target Node, oldValue string) {
	record := &MutationRecord{
		Type:     "characterData",
		Target:   target,
		OldValue: oldValue,
	}
	notifyObservers(target, record)
}

func notifyObservers(target Node, record *MutationRecord) {
	notified := map[*MutationObserver]bool{}
	for node := target; node != nil; node = node.ParentNode() {
		regs := observerRegistry[node]
		for _, reg := range regs {
			if notified[reg.observer] {
				continue
			}
			if node != target && !reg.options.Subtree {
				continue
			}
			if !matchesOptions(record, reg.options) {
				continue
			}
			// Strip OldValue if not requested
			r := record
			if record.Type == "attributes" && !reg.options.AttributeOldValue {
				r = &MutationRecord{
					Type:          record.Type,
					Target:        record.Target,
					AttributeName: record.AttributeName,
				}
			}
			if record.Type == "characterData" && !reg.options.CharacterDataOldValue {
				r = &MutationRecord{
					Type:   record.Type,
					Target: record.Target,
				}
			}
			reg.observer.records = append(reg.observer.records, r)
			notified[reg.observer] = true
			enqueuePending(reg.observer)
		}
	}
}

func matchesOptions(record *MutationRecord, opts MutationObserverInit) bool {
	switch record.Type {
	case "childList":
		return opts.ChildList
	case "attributes":
		if !opts.Attributes {
			return false
		}
		if len(opts.AttributeFilter) > 0 {
			for _, f := range opts.AttributeFilter {
				if f == record.AttributeName {
					return true
				}
			}
			return false
		}
		return true
	case "characterData":
		return opts.CharacterData
	}
	return false
}
