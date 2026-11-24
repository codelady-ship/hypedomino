#import <UIKit/UIKit.h>

extern "C" int HypeDominoReduceMotion()
{
    return UIAccessibilityIsReduceMotionEnabled() ? 1 : 0;
}
