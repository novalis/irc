package main

type NumericMessage struct {
	number   int
	template string
}

/* These messages come straight from the RFC */

var (
	ERR_NOSUCHNICK       = NumericMessage{401, "%s :No such nick/channel"}
	ERR_NOSUCHSERVER     = NumericMessage{402, "%s :No such server"}
	ERR_NOSUCHCHANNEL    = NumericMessage{403, "%s :No such channel"}
	ERR_CANNOTSENDTOCHAN = NumericMessage{404, "%s :Cannot send to channel"}
	ERR_TOOMANYCHANNELS  = NumericMessage{405, "%s :You have joined too many channels"}
	ERR_WASNOSUCHNICK    = NumericMessage{406, "%s :There was no such nickname"}
	ERR_TOOMANYTARGETS   = NumericMessage{407, "%s :Duplicate recipients. No message delivered"}
	ERR_NOORIGIN         = NumericMessage{409, ":No origin specified"}
	ERR_NORECIPIENT      = NumericMessage{411, ":No recipient given (%s)"}
	ERR_NOTEXTTOSEND     = NumericMessage{412, ":No text to send"}
	ERR_NOTOPLEVEL       = NumericMessage{413, "%s :No toplevel domain specified"}
	ERR_WILDTOPLEVEL     = NumericMessage{414, "%s :Wildcard in toplevel domain"}
	ERR_UNKNOWNCOMMAND   = NumericMessage{421, "%s :Unknown command"}
	ERR_NOMOTD           = NumericMessage{422, ":MOTD File is missing"}
	ERR_NOADMININFO      = NumericMessage{423, "%s :No administrative info available"}
	ERR_FILEERROR        = NumericMessage{424, ":File error doing %s on %s"}
	ERR_NONICKNAMEGIVEN  = NumericMessage{431, ":No nickname given"}
	ERR_ERRONEUSNICKNAME = NumericMessage{432, "%s :Erroneous nickname"} //sic
	ERR_NICKNAMEINUSE    = NumericMessage{433, "%s :Nickname is already in use"}
	ERR_NICKCOLLISION    = NumericMessage{436, "%s :Nickname collision KILL"}
	ERR_USERNOTINCHANNEL = NumericMessage{441, "%s %s :They aren't on that channel"}
	ERR_NOTONCHANNEL     = NumericMessage{442, "%s :You're not on that channel"}
	ERR_USERONCHANNEL    = NumericMessage{443, "%s %s :is already on channel"}
	ERR_NOLOGIN          = NumericMessage{444, "%s :User not logged in"}
	ERR_SUMMONDISABLED   = NumericMessage{445, ":SUMMON has been disabled"}
	ERR_USERSDISABLED    = NumericMessage{446, ":USERS has been disabled"}
	ERR_NOTREGISTERED    = NumericMessage{451, ":You have not registered"}
	ERR_NEEDMOREPARAMS   = NumericMessage{461, "%s :Not enough parameters"}
	ERR_ALREADYREGISTRED = NumericMessage{462, ":You may not reregister"}
	ERR_NOPERMFORHOST    = NumericMessage{463, ":Your host isn't among the privileged"}
	ERR_PASSWDMISMATCH   = NumericMessage{464, ":Password incorrect"}
	ERR_YOUREBANNEDCREEP = NumericMessage{465, ":You are banned from this server"}
	ERR_KEYSET           = NumericMessage{467, "%s :Channel key already set"}
	ERR_CHANNELISFULL    = NumericMessage{471, "%s :Cannot join channel (+l)"}
	ERR_UNKNOWNMODE      = NumericMessage{472, "%s :is unknown mode char to me"}
	ERR_INVITEONLYCHAN   = NumericMessage{473, "%s :Cannot join channel (+i)"}
	ERR_BANNEDFROMCHAN   = NumericMessage{474, "%s :Cannot join channel (+b)"}
	ERR_BADCHANNELKEY    = NumericMessage{475, "%s :Cannot join channel (+k)"}
	ERR_NOPRIVILEGES     = NumericMessage{481, ":Permission Denied- You're not an IRC operator"}
	ERR_CHANOPRIVSNEEDED = NumericMessage{482, "%s :You're not channel operator"}
	ERR_CANTKILLSERVER   = NumericMessage{483, ":You cant kill a server!"}

	ERR_RESTRICTED       = NumericMessage{484, ":Your connection is restricted!"}
	ERR_NOOPERHOST       = NumericMessage{491, ":No O-lines for your host"}
	ERR_UMODEUNKNOWNFLAG = NumericMessage{501, ":Unknown MODE flag"}
	ERR_USERSDONTMATCH   = NumericMessage{502, ":Cannot change mode for other users"}

	RPL_WELCOME = NumericMessage{1, "Welcome to the Internet Relay Network %s!%s@%s"}

	RPL_YOURHOST = NumericMessage{2, "Your host is %s, running version %s"}
	RPL_CREATED  = NumericMessage{3, "This server was created %s"}
	RPL_MYINFO   = NumericMessage{4, "%s %s %s %s"}

	RPL_BOUNCE = NumericMessage{5, "Try server %s, port %s"}

	RPL_USERHOST        = NumericMessage{302, ":[%s{%s%s}]"}
	RPL_ISON            = NumericMessage{303, ":[%s {%s%s}]"}
	RPL_AWAY            = NumericMessage{301, "%s :%s"}
	RPL_UNAWAY          = NumericMessage{305, ":You are no longer marked as being away"}
	RPL_NOWAWAY         = NumericMessage{306, ":You have been marked as being away"}
	RPL_WHOISUSER       = NumericMessage{311, "%s %s %s * :%s"}
	RPL_WHOISSERVER     = NumericMessage{312, "%s %s :%s"}
	RPL_WHOISOPERATOR   = NumericMessage{313, "%s :is an IRC operator"}
	RPL_WHOISIDLE       = NumericMessage{317, "%s %s :seconds idle"}
	RPL_ENDOFWHOIS      = NumericMessage{318, "%s :End of /WHOIS list"}
	RPL_WHOISCHANNELS   = NumericMessage{319, "%s :{[@|+]%s%s}"}
	RPL_WHOWASUSER      = NumericMessage{314, "%s %s %s * :%s"}
	RPL_ENDOFWHOWAS     = NumericMessage{369, "%s :End of WHOWAS"}
	RPL_LISTSTART       = NumericMessage{321, "Channel :Users Name"}
	RPL_LIST            = NumericMessage{322, "%s %s :%s"}
	RPL_LISTEND         = NumericMessage{323, ":End of /LIST"}
	RPL_CHANNELMODEIS   = NumericMessage{324, "%s %s %s"}
	RPL_NOTOPIC         = NumericMessage{331, "%s :No topic is set"}
	RPL_TOPIC           = NumericMessage{332, "%s :%s"}
	RPL_INVITING        = NumericMessage{341, "%s %s"}
	RPL_SUMMONING       = NumericMessage{342, "%s :Summoning user to IRC"}
	RPL_VERSION         = NumericMessage{351, "%s.%s %s :%s"}
	RPL_WHOREPLY        = NumericMessage{352, "%s %s %s %s %s %s :%s %s"}
	RPL_ENDOFWHO        = NumericMessage{315, "%s :End of /WHO list"}
	RPL_NAMREPLY        = NumericMessage{353, "%s %s :%s"}
	RPL_ENDOFNAMES      = NumericMessage{366, "%s :End of /NAMES list"}
	RPL_LINKS           = NumericMessage{364, "%s %s :%s %s"}
	RPL_ENDOFLINKS      = NumericMessage{365, "%s :End of /LINKS list"}
	RPL_BANLIST         = NumericMessage{367, "%s %s"}
	RPL_ENDOFBANLIST    = NumericMessage{368, "%s :End of channel ban list"}
	RPL_INFO            = NumericMessage{371, ":%s"}
	RPL_ENDOFINFO       = NumericMessage{374, ":End of /INFO list"}
	RPL_MOTDSTART       = NumericMessage{375, ":- %s Message of the day - "}
	RPL_MOTD            = NumericMessage{372, ":- %s"}
	RPL_ENDOFMOTD       = NumericMessage{376, ":End of /MOTD command"}
	RPL_YOUREOPER       = NumericMessage{381, ":You are now an IRC operator"}
	RPL_REHASHING       = NumericMessage{382, "%s :Rehashing"}
	RPL_TIME            = NumericMessage{391, "%s :%s"}
	RPL_USERSSTART      = NumericMessage{392, ":UserID Terminal Host"}
	RPL_USERS           = NumericMessage{393, ":%-8s %-9s %-8s"}
	RPL_ENDOFUSERS      = NumericMessage{394, ":End of users"}
	RPL_NOUSERS         = NumericMessage{395, ":Nobody logged in"}
	RPL_TRACELINK       = NumericMessage{200, "Link %s %s %s"}
	RPL_TRACECONNECTING = NumericMessage{201, "Try. %s %s"}
	RPL_TRACEHANDSHAKE  = NumericMessage{202, "H.S. %s %s"}
	RPL_TRACEUNKNOWN    = NumericMessage{203, "???? %s [%s]"}
	RPL_TRACEOPERATOR   = NumericMessage{204, "Oper %s %s"}
	RPL_TRACEUSER       = NumericMessage{205, "User %s %s"}
	RPL_TRACESERVER     = NumericMessage{206, "Serv %s %sS %sC %s %s@%s"}
	RPL_TRACENEWTYPE    = NumericMessage{208, "%s 0 %s"}
	RPL_TRACELOG        = NumericMessage{261, "File %s %s"}
	RPL_STATSLINKINFO   = NumericMessage{211, "%s %s %s %s %s %s %s"}
	RPL_STATSCOMMANDS   = NumericMessage{212, "%s %s"}
	RPL_STATSCLINE      = NumericMessage{213, "C %s * %s %s %s"}
	RPL_STATSNLINE      = NumericMessage{214, "N %s * %s %s %s"}
	RPL_STATSILINE      = NumericMessage{215, "I %s * %s %s %s"}
	RPL_STATSKLINE      = NumericMessage{216, "K %s * %s %s %s"}
	RPL_STATSYLINE      = NumericMessage{218, "Y %s %s %s %s"}
	RPL_ENDOFSTATS      = NumericMessage{219, "%s :End of /STATS report"}
	RPL_STATSLLINE      = NumericMessage{241, "L %s * %s %s"}
	RPL_STATSUPTIME     = NumericMessage{242, ":Server Up %d days %d:%02d:%02d"}
	RPL_STATSOLINE      = NumericMessage{243, "O %s * %s"}
	RPL_STATSHLINE      = NumericMessage{244, "H %s * %s"}
	RPL_UMODEIS         = NumericMessage{221, "%s"}
	RPL_LUSERCLIENT     = NumericMessage{251, ":There are %s users and %s invisible on %s servers"}
	RPL_LUSEROP         = NumericMessage{252, "%s :operator(s) online"}
	RPL_LUSERUNKNOWN    = NumericMessage{253, "%s :unknown connection(s)"}
	RPL_LUSERCHANNELS   = NumericMessage{254, "%s :channels formed"}
	RPL_LUSERME         = NumericMessage{255, ":I have %s clients and %s servers"}
	RPL_ADMINME         = NumericMessage{256, "%s :Administrative info"}
	RPL_ADMINLOC1       = NumericMessage{257, ":%s"}
	RPL_ADMINLOC2       = NumericMessage{258, ":%s"}
	RPL_ADMINEMAIL      = NumericMessage{259, ":%s"}
)
