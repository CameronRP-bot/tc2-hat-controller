// DBus service for controlling the trap. This lets other services send requests,
// such as restarting the trap, while trap-control is running and has the serial port open.

package comms

import (
	"errors"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	trapDbusName = "org.cacophony.trap"
	trapDbusPath = "/org/cacophony/trap"
)

type trapService struct {
	messenger *TrapMessenger
}

// trapServiceConn is the bus the trap service is on, once it has started, for sending
// signals from messages the trap sends on its own.
var trapServiceConn atomic.Pointer[dbus.Conn]

// startTrapService exports the trap service onto the system bus.
// Requests are sent to the trap through the given messenger, which serializes them
// with the messages being sent by the trap control loop.
func startTrapService(messenger *TrapMessenger) error {
	log.Info("Starting trap service")
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	reply, err := conn.RequestName(trapDbusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return errors.New("name already taken")
	}

	s := &trapService{
		messenger: messenger,
	}
	if err := conn.Export(s, trapDbusPath, trapDbusName); err != nil {
		return err
	}
	if err := conn.Export(genTrapIntrospectable(s), trapDbusPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return err
	}
	trapServiceConn.Store(conn)
	return nil
}

// emitTrapStatus sends a StatusChanged signal with the trap's status, as JSON (see
// TrapMessenger.Status), when the trap reports that it has changed. Changes from before the
// service has started are dropped, the Status method gets the current one.
func emitTrapStatus(status string) {
	conn := trapServiceConn.Load()
	if conn == nil {
		return
	}
	if err := conn.Emit(trapDbusPath, trapDbusName+".StatusChanged", status); err != nil {
		log.Errorf("Failed to send the trap's status over DBus: %v", err)
	}
}

// Status returns what the trap is doing, as JSON (see TrapMessenger.Status).
// It isn't logged, as it is polled while the trap page is open.
func (s *trapService) Status() (string, *dbus.Error) {
	status, err := s.messenger.Status()
	if err != nil {
		return "", trapDbusErr(err)
	}
	return status, nil
}

// Restart restarts the trap.
func (s *trapService) Restart() *dbus.Error {
	log.Info("Restarting trap from DBus request")
	if err := s.messenger.Restart(); err != nil {
		log.Errorf("Failed to restart trap: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// Stop asks the trap to stop what it is doing, putting it into manual mode.
func (s *trapService) Stop() *dbus.Error {
	log.Info("Stopping trap from DBus request")
	if err := s.messenger.Stop(); err != nil {
		log.Errorf("Failed to stop trap: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// ReleaseSpool asks the trap to release the spool, putting it into manual mode.
func (s *trapService) ReleaseSpool() *dbus.Error {
	log.Info("Releasing spool from DBus request")
	if err := s.messenger.ReleaseSpool(); err != nil {
		log.Errorf("Failed to release spool: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// ResetSpool asks the trap to reset the spool, putting it into manual mode.
func (s *trapService) ResetSpool() *dbus.Error {
	log.Info("Resetting spool from DBus request")
	if err := s.messenger.ResetSpool(); err != nil {
		log.Errorf("Failed to reset spool: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// OpenDoor asks the trap to open one of its ratchet doors, putting it into manual mode.
func (s *trapService) OpenDoor(door int32) *dbus.Error {
	log.Infof("Opening door %d from DBus request", door)
	if err := s.messenger.OpenDoor(int(door)); err != nil {
		log.Errorf("Failed to open door %d: %v", door, err)
		return trapDbusErr(err)
	}
	return nil
}

// CloseDoor asks the trap to close one of its ratchet doors, putting it into manual mode.
func (s *trapService) CloseDoor(door int32) *dbus.Error {
	log.Infof("Closing door %d from DBus request", door)
	if err := s.messenger.CloseDoor(int(door)); err != nil {
		log.Errorf("Failed to close door %d: %v", door, err)
		return trapDbusErr(err)
	}
	return nil
}

// CageHold asks the trap to keep holding the animal in its cage for longer.
func (s *trapService) CageHold() *dbus.Error {
	log.Info("Holding the animal in the cage from DBus request")
	if err := s.messenger.CageHold(); err != nil {
		log.Errorf("Failed to hold the animal in the cage: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// CageRelease asks the trap to let the animal out of its cage.
func (s *trapService) CageRelease() *dbus.Error {
	log.Info("Releasing the animal from the cage from DBus request")
	if err := s.messenger.CageRelease(); err != nil {
		log.Errorf("Failed to release the animal from the cage: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// CageKill asks the trap to open its cage to the kill mechanism.
func (s *trapService) CageKill() *dbus.Error {
	log.Info("Opening the cage to the kill mechanism from DBus request")
	if err := s.messenger.CageKill(); err != nil {
		log.Errorf("Failed to open the cage to the kill mechanism: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// CageWait asks the trap to wait longer for the animal to leave its cage.
func (s *trapService) CageWait() *dbus.Error {
	log.Info("Waiting longer for the animal to leave the cage from DBus request")
	if err := s.messenger.CageWait(); err != nil {
		log.Errorf("Failed to wait longer for the animal to leave the cage: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// CageEscaped tells the trap the animal has left its cage.
func (s *trapService) CageEscaped() *dbus.Error {
	log.Info("Marking the animal in the cage as escaped from DBus request")
	if err := s.messenger.CageEscaped(); err != nil {
		log.Errorf("Failed to mark the animal in the cage as escaped: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

// CageKilled tells the trap the animal in its cage has been killed.
func (s *trapService) CageKilled() *dbus.Error {
	log.Info("Marking the animal in the cage as killed from DBus request")
	if err := s.messenger.CageKilled(); err != nil {
		log.Errorf("Failed to mark the animal in the cage as killed: %v", err)
		return trapDbusErr(err)
	}
	return nil
}

func genTrapIntrospectable(v interface{}) introspect.Introspectable {
	node := &introspect.Node{
		Interfaces: []introspect.Interface{{
			Name:    trapDbusName,
			Methods: introspect.Methods(v),
			Signals: []introspect.Signal{{
				Name: "StatusChanged",
				Args: []introspect.Arg{{Name: "status", Type: "s"}},
			}},
		}},
	}
	return introspect.NewIntrospectable(node)
}

func trapDbusErr(err error) *dbus.Error {
	if err == nil {
		return nil
	}
	return &dbus.Error{
		Name: trapDbusName + "." + getTrapCallerName(),
		Body: []interface{}{err.Error()},
	}
}

func getTrapCallerName() string {
	fpcs := make([]uintptr, 1)
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return ""
	}
	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return ""
	}
	funcNames := strings.Split(caller.Name(), ".")
	return funcNames[len(funcNames)-1]
}
