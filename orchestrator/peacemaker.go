package main

// TODO: bring peace among nodes that contest a certain record id
// this may happen when a new node is added, with already loaded records.
// peacemaker will find collision and apply a fix:
//  - leave the record only on one node when they are the same
//  - change the record id when different
