#import <UIKit/UIKit.h>

extern "C" void HypeDominoPlayHaptic(int cue)
{
    switch (cue)
    {
        case 1:
            [[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleLight] impactOccurred];
            break;
        case 2:
            [[[UIImpactFeedbackGenerator alloc] initWithStyle:UIImpactFeedbackStyleMedium] impactOccurred];
            break;
        case 3:
            [[[UINotificationFeedbackGenerator alloc] init] notificationOccurred:UINotificationFeedbackTypeSuccess];
            break;
    }
}
