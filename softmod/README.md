# Soft mod files for factorio operator ( factop )

* https://github.com/mlctrez/factop/tree/master/softmod

## contents

* factop/*.lua - libraries passed to the built-in factorio event_handler code
* locale/\<lang>/*.cfg - localization files
* img/* - any images bundled with the softmod
* softmod.go - golang code that creates the softmod zip payload
* controlHeader.lua - used by softmod.go to create the final control.lua