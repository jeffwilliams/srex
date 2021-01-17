# srex

srex is a command-line tool to match sections of files using Rob Pike's [Structural Regular Expressions](http://doc.cat-v.org/bell_labs/structural_regexps/). It implements the commands with a similar syntax as the [Sam](http://sam.cat-v.org/) editor, but only implements some of the commands. Notably omitted are the ones that modify the file -- this tool only prints out matches.

The primary motivation for this tool is to handle a use-case that occasionally comes up when using grep but isn't easy to handle: when the input consists of multi-line records and you want to select matching records, but you want to select the entire record not just the matching line in the record. The workaround using grep usually involves tweaking the -A and -B grep options, but doesn't really handle the case well.

The regular expression syntax is that from the Go [regexp package](https://golang.org/pkg/regexp/syntax/).

# Command syntax

The following Sam-like commands are supported:

   * **x/pattern/**		Extract: loop over each match of the regular expression pattern and run the subsequent command on the matched text
   * **y/pattern/**      Compliment of x: loop over each piece between the matches of pattern and run subsequent commands
   * **g/pattern/**      Run the subsequent command only if the text matches the pattern. This is a conditional.
   * **v/pattern/**      Compliment of g: Only run the subsequent command if the text does not match the pattern.
   * **p**            Print the matching text. This is the default command so may be omitted
   * **=**          Print the line numbers of the start and end of the match
   
There are also some commands not supported in sam: 

   * **n[indexes]**	Only select the ranges with the specified indexes. Valid values for indexes include:
   
      1. N   a single number selects the range N only. Ranges are counted starting from 0. If N is negative it specifies counts from the last element instead
      2. N:M  select ranges who's index is >= N and <= M. M may be negative.
      3. N:     select ranges who's index is >= N

# Usage

Invoke srex like so:

		srex <file> <commands>
		
The commands must all be embedded in a single command-line argument. This means you'll generally surround them with single quotes. For example: 

		srex software.log 'x/start(.|\n)*?end/ g/debug/'

# Examples

To illustrate the use-case described above we'll take an input file and run some matches. We'll use this event-history output of a show command from a Cisco switch taken from [here](https://www.cisco.com/c/m/en_us/techdoc/dc/reference/cli/n5k/commands/show-routing-ip-multicast-event-history.html) as the input file named 'example':

    Msg events for MRIB Process
    1) Event:E_DEBUG, length:38, at 932956 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: transient thread created
    
    2) Event:E_DEBUG, length:38, at 932269 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: create transcient thread
    
    3) Event:E_DEBUG, length:75, at 932264 usecs after Sat Apr 12 09:09:41 2008
        [100] : comp-mts-rx opc - from sap 3210 cmd mrib_internal_event_hist_command
    4) Event:E_MTS_RX, length:60, at 362578 usecs after Sat Apr 12 09:08:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F217E, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F217B, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    5) Event:E_MTS_RX, length:60, at 352493 usecs after Sat Apr 12 09:07:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F188B, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F1888, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    6) Event:E_MTS_RX, length:60, at 342641 usecs after Sat Apr 12 09:06:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0DF0, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F0DED, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    7) Event:E_MTS_RX, length:60, at 332954 usecs after Sat Apr 12 09:05:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0493, Ret:SUCCESS
    <--Output truncated-->
    switch(config)#
    
Notice that the history consists of a series of multi-line records. Each begins with 'N) Event:' and is followed by one or more space-indented lines. We can select each of these records using:

    srex example 'x/\d+\) Event:.*\n( +.*\n)*/'
    
Let's break down that regular expression inside the x//. The first part of the regular expression `\d+\) Event:.*\n` matches the first line, and the next part `( +.*\n)*` matches the indented lines that follow the first as part of the record. This gives the output:

    1) Event:E_DEBUG, length:38, at 932956 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: transient thread created
    2) Event:E_DEBUG, length:38, at 932269 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: create transcient thread
    3) Event:E_DEBUG, length:75, at 932264 usecs after Sat Apr 12 09:09:41 2008
        [100] : comp-mts-rx opc - from sap 3210 cmd mrib_internal_event_hist_command
    4) Event:E_MTS_RX, length:60, at 362578 usecs after Sat Apr 12 09:08:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F217E, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F217B, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    5) Event:E_MTS_RX, length:60, at 352493 usecs after Sat Apr 12 09:07:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F188B, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F1888, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    6) Event:E_MTS_RX, length:60, at 342641 usecs after Sat Apr 12 09:06:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0DF0, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F0DED, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    7) Event:E_MTS_RX, length:60, at 332954 usecs after Sat Apr 12 09:05:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0493, Ret:SUCCESS

This looks basically like the input file, since srex is just printing the matches verbatim. Let's use a separator to clearly see where the records begin and end. The `-s` or `--separator` argument specifies a string to print between each record:

    srex -s '----------\n' example 'x/\d+\) Event:.*\n( +.*\n)*/'
    
which gives us:

    1) Event:E_DEBUG, length:38, at 932956 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: transient thread created
    ----------
    2) Event:E_DEBUG, length:38, at 932269 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: create transcient thread
    ----------
    3) Event:E_DEBUG, length:75, at 932264 usecs after Sat Apr 12 09:09:41 2008
        [100] : comp-mts-rx opc - from sap 3210 cmd mrib_internal_event_hist_command
    ----------
    4) Event:E_MTS_RX, length:60, at 362578 usecs after Sat Apr 12 09:08:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F217E, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F217B, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    ----------
    5) Event:E_MTS_RX, length:60, at 352493 usecs after Sat Apr 12 09:07:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F188B, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F1888, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    ----------
    6) Event:E_MTS_RX, length:60, at 342641 usecs after Sat Apr 12 09:06:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0DF0, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F0DED, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    ----------
    7) Event:E_MTS_RX, length:60, at 332954 usecs after Sat Apr 12 09:05:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0493, Ret:SUCCESS

Now let's only select the records that have the opcode related to ipv4 route statistics. For that we add a second command after the `x//` command to select specific records: the `g//` command:

    srex  example 'x/\d+\) Event:.*\n( +.*\n)*/ g/MTS_OPC_MFDM_V4_ROUTE_STATS/'

This command gives:

    4) Event:E_MTS_RX, length:60, at 362578 usecs after Sat Apr 12 09:08:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F217E, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F217B, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    5) Event:E_MTS_RX, length:60, at 352493 usecs after Sat Apr 12 09:07:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F188B, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F1888, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    6) Event:E_MTS_RX, length:60, at 342641 usecs after Sat Apr 12 09:06:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0DF0, Ret:SUCCESS
        Src:0x00000101/214, Dst:0x00000101/1203, Flags:None
        HA_SEQNO:0X00000000, RRtoken:0x000F0DED, Sync:NONE, Payloadsize:148
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
    7) Event:E_MTS_RX, length:60, at 332954 usecs after Sat Apr 12 09:05:51 2008
        [RSP] Opc:MTS_OPC_MFDM_V4_ROUTE_STATS(75785), Id:0X000F0493, Ret:SUCCESS

Note that the entire matching record is printed, not just the line with the opcode. Because the g command is placed after the x command it only applies to records that the x command outputs. You can think of the commands as parts of a pipeline. 

Say instead we wanted to find all records that are _not_ about that opcode. We could use:

    srex  example 'x/\d+\) Event:.*\n( +.*\n)*/ v/MTS_OPC_MFDM_V4_ROUTE_STATS/'
    
to get:

    1) Event:E_DEBUG, length:38, at 932956 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: transient thread created
    2) Event:E_DEBUG, length:38, at 932269 usecs after Sat Apr 12 09:09:41 2008
        [100] : nvdb: create transcient thread
    3) Event:E_DEBUG, length:75, at 932264 usecs after Sat Apr 12 09:09:41 2008
        [100] : comp-mts-rx opc - from sap 3210 cmd mrib_internal_event_hist_command

Say we wanted to again select all the records that have the opcode related to ipv4 route statistics as before, but now we want to only print their payloads, we could use an x// command to select the records, a g// command to filter for the opcode we care about, and then use a final x// command to extract the payload fields from the records:

    srex  example 'x/\d+\) Event:.*\n( +.*\n)*/ g/MTS_OPC_MFDM_V4_ROUTE_STATS/ x/(\n\s*Payload.*)|(\n\s*0x.*)/'

which gives:

        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
        Payload:
        0x0000:  01 00 00 00 05 00 01 00 00 04 00 00 00 00 00 00
        
# TODO

Document an example for using the n[indexes] command.