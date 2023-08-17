#!/usr/bin/expect -f

spawn telescope transpile
while {1} {
    expect {
        ".*" { send "\r" }
        eof { break }
    }
}