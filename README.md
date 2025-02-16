# go-ARI-voice2SMS-application-for-PBXWare-using-OpenAI-transcription-and-translation

ARI app specifically adjusted to the Bicom PBXWare environment, written in go due to hosted PBXWare constraints.

OpenAI API for transcription & translation, Telnyx API for SMS delivery

Has turned into Swiss Army knife for OpenAI interactions

Now can also playbck translation as well as do a signle question and answer to ChatGPT in different languages, input & output.

        Sustained chat is now included.
        I am now integrating embeddings and CRM connectivity to be able to have a meaningful conversation.
        I will post this soon.

        My sticking point is now being able to trasfer the call to an Extension, IVR, Queue, etc.
                I have tried many ways, creating channels, bridges, and both, without success.
                Until I can achieve this, all this is no more than an application that can chat, translate & SMS . . .

Can be customized from command line, to allow running multiple simultaneous versions for different purposes

Output language and voice can be defined at command line (language can be overrriden using PBXWare 'Replace Caller ID' as : lang=german)

Can prompt for a callback number to be entered as DTMF (or # for as CID) or skipped with nocallbacknum

Various recordings can be specified or skipped.
        Current usage, we only use the 'goodby_sound_file' and use IVRs & RGs for the rest, basically an SMS mailbox

Can be verbose or silent (to stdout)

Can be set to simply transcribe or translate to english and include the original transcription

Tenant can be 1 for Call Centre or Business editions

Email can be provided to also send email with attachment, since doing the attachment with MMS seems insecure (exposed on public webserver)

UUsage :

        voiceSMS IP User Pass AppName Tenant ["SMS Name"|-] [SMS_To|List|-] [SMS_From|-] [callbacknum|nocallbacknum|-] [en|fr|indonesian|-] [transcribe|translate|transback|chat|-] [alloy|echo|fable|onyx|nova|shimmer||-] [email_To|List|-] [LogFile|-] [DTMF_Instructions_File|-] [Recording_Instructions_File|-] [Goodbye_Sound|-] [verbose|silent|-]

        voiceSMS 127.0.0.1 ariuser aripass ariapp 999 "Sender Name" +15555555555 +15556667777 callbacknum - transcribe shimmer email@domain.suf /var/log/ari_log.txt dtmf_instuctions recording_instructions goodbye_sound_file silent

        voiceSMS 127.0.0.1 ariuser aripass ariapp 999 - - - - - - - - - - - - -

