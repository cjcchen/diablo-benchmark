#pragma once

#include <string.h>
#include <stdint.h>

int SendTransaction(uint64_t uid, 
    const char * from, 
    const char * to,
    uint64_t amount);
