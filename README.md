# go-ARI-voice2SMS-application-for-PBXWare-using-OpenAI-transcription-and-translation

ARI app specifically adjusted to the Bicom PBXWare environment, written in go due to hosted PBXWare constraints.

OpenAI API for transcription & translation, Telnyx API for SMS delivery

Can be customized from command line, to allow running multiple simultaneous versions for different purposes

Can prompt for a callback number to be entered as DTMF (or # for as CID) or skipped with nocallbacknum

Various recordings can be specified or skipped.
        Current usage, we only use the 'goodby_sound_file' and use IVRs & RGs for the rest, basically an SMS mailbox

Can be verbose or silent (to stdout)

Can be set to simply transcribe or translate to english and include the original transcription

Tenant can be 1 for Call Centre or Business editions

Usage :

        voiceSMS IP User Pass AppName Tenant ["SMS Name"|-] [SMS_To|List|-] [SMS_From|-] [callbacknum|nocallbacknum|-] [transcribe|translate|-] [LogFile|-] [DTMF_Instructions_File|-] [Recording_Instructions_File|-] [Goodbye_Sound|-] [verbose|silent|-]

        voiceSMS 127.0.0.1 ariuser aripass ariapp 999 "Sender Name" +15555555555 +15556667777 callbacknum transcribe /var/log/ari_log.txt dtmf_instuctions recording_instructions goodbye_sound_file silent

        voiceSMS 127.0.0.1 ariuser aripass ariapp 999 - - - - - - - - - -

