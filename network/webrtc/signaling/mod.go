// SIGNALING. Allows both room and simplest signaling.
// Signaling is the process of relaying information between peers in a webrtc context.
//   Namely information about offers, answers and ice-candidates
// Simplest Signaling:
//   To a known user, this server can relay that information.
//   It might be required to additionally add peer finding functionality. I.e. a query of who is currently in a room.
//      And broadcast a notify if a new user connects.
//      This was not required for the concrete problem at hand that this util-library aimed to solve
//      But it is trivial to add.
// Room signaling is just simplest signaling, with virtual servers.
//   For how rooms are created, destroyed and function in general, see the room_forwarding_*.go files in wsclientable
package signaling
