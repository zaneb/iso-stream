# iso-stream

The goal of this test program is to inject the ignition file into an rhcos iso while streaming

POC a Reader implementation that will determine the ignition reserved area location while reading the iso then inject the ignition content into the stream in the correct location
