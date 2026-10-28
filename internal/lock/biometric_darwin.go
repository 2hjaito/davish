//go:build darwin

package lock

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework LocalAuthentication
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>

static int davishBiometricAvailable(void) {
	LAContext *ctx = [[LAContext alloc] init];
	return [ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:nil] ? 1 : 0;
}

// Shows the Touch ID sheet and waits for it: 1 when it passed, 0 otherwise.
static int davishBiometricAuthenticate(const char *reason) {
	LAContext *ctx = [[LAContext alloc] init];
	// No "Enter Password" button: the fallback is Davish's own password.
	ctx.localizedFallbackTitle = @"";
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	__block int ok = 0;
	[ctx evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
	    localizedReason:[NSString stringWithUTF8String:reason]
	              reply:^(BOOL success, NSError *error) {
		ok = success ? 1 : 0;
		dispatch_semaphore_signal(done);
	}];
	dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
	return ok;
}
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"
)

// NewBiometric returns Touch ID through LocalAuthentication.
func NewBiometric() Biometric { return touchID{} }

type touchID struct{}

func (touchID) Available() bool { return C.davishBiometricAvailable() == 1 }

// Authenticate blocks until the user passes or cancels the Touch ID sheet;
// ctx cannot cancel a sheet the OS is showing.
func (touchID) Authenticate(_ context.Context, reason string) error {
	cs := C.CString(reason)
	defer C.free(unsafe.Pointer(cs))
	if C.davishBiometricAuthenticate(cs) != 1 {
		return errors.New("touch id did not pass")
	}
	return nil
}
