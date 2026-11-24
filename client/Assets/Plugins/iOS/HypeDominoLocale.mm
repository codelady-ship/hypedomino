#import <Foundation/Foundation.h>
#include <string.h>

extern "C" char* HypeDominoPreferredLanguages()
{
    NSString* joined = [[NSLocale preferredLanguages] componentsJoinedByString:@","];
    return strdup(joined.UTF8String);
}
