#import <Foundation/Foundation.h>

extern "C" int HypeDominoLowPowerMode()
{
    return [NSProcessInfo processInfo].isLowPowerModeEnabled ? 1 : 0;
}
