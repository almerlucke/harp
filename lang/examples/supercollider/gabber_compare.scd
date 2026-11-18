// The reference for 11_gabber_rave.harp (and 9_gabber_kick.harp, 10_hoover.harp): the Pbinds of "60Hz Gabber Rave 1995" played by
// SuperCollider, offline (20 seconds, 44100 Hz, 32 bit float), as kick.wav (the kick), hoover.wav and full.wav (both).
//   sclang -D gabber_compare.scd /some/folder
(
Server.program = "/home/almer/.local/supercollider/bin/scsynth";
Score.program = Server.program;
~dir = thisProcess.argv.last ?? "."; // the folder the wav files and the synth defs are written to: sclang gabber_compare.scd /some/folder
try {
~defs = List.new;
~k = SynthDef(\gabberkick, {
    var snd, freq, high, lfo;
    freq = \freq.kr(440) * (Env.perc(0.001, 0.08, curve: -1).ar * 48 * \bend.kr(1)).midiratio;
    snd = Saw.ar(freq);
    snd = (snd * 100).tanh + ((snd.sign - snd) * -8.dbamp);
    high = HPF.ar(snd, 300);
    lfo = SinOsc.ar(8, [0, 0.5pi]).range(0, 0.01);
    high = high.dup(2) + (DelayC.ar(high, 0.01, lfo) * -2.dbamp);
    snd = LPF.ar(snd, 100).dup(2) + high;
    snd = RLPF.ar(snd, 7000, 2);
    snd = BPeakEQ.ar(snd, \ffreq.kr(3000) * XLine.kr(1, 0.8, 0.3), 0.5, 15);
    snd = snd * Env.asr(0.001, 1, 0.05).ar(2, \gate.kr(1));
    Out.ar(\out.kr(0), snd * \amp.kr(0.1));
});
~h = SynthDef(\hoover, {
    var snd, freq, bw, delay, decay;
    freq = \freq.kr(440);
    freq = freq * Env([-5, 6, 0], [0.1, 1.7], [\lin, -4]).kr.midiratio;
    bw = 1.035;
    snd = { DelayN.ar(Saw.ar(freq * ExpRand(bw, 1 / bw)) + Saw.ar(freq * 0.5 * ExpRand(bw, 1 / bw)), 0.01, Rand(0, 0.01)) }.dup(20);
    snd = (Splay.ar(snd) * 3).atan;
    snd = snd * Env.asr(0.01, 1.0, 1.0).kr(0, \gate.kr(1));
    snd = FreeVerb2.ar(snd[0], snd[1], 0.3, 0.9);
    snd = snd * Env.asr(0, 1.0, 4, 6).kr(2, \gate.kr(1));
    Out.ar(\out.kr(0), snd * \amp.kr(0.1));
});
~k.store(\global, ~dir); ~h.store(\global, ~dir);
thisThread.randSeed = 1;
~durations = [1, 1, 1, 1, 3/4, 1/4, 1/2, 3/4, 1/4, 1/2];
~pk = Pbind(*[
        instrument: \gabberkick, amp: -23.dbamp, freq: 60, legato: 0.8,
        ffreq: Pseq((0..(~durations.size * 4 - 1)).normalize, inf).linexp(0, 1, 100, 4000),
        dur: Pseq(~durations, inf),
        bend: Pfuncn({ |x| if(x < (1/2), 0.4, 1) }, inf) <> Pkey(\dur),
    ]);
~ph = Pbind(*[
        instrument: \hoover, amp: -20.dbamp, midinote: 74,
        dur: ~durations.sum * 2, sustain: 7,
    ]);

~render = { |pat, secs, file|
    var sc, cmds, done = Condition.new, k = 60 / 210;
    sc = pat.asScore(secs / k);
    cmds = [[0.0, ['/d_recv', ~k.asBytes]], [0.0, ['/d_recv', ~h.asBytes]]] ++ sc.score.collect { |e| [e[0] * k] ++ e[1..] };
    cmds = cmds.select { |e| e[1][0] != '/g_new' } ++ [[secs + 1, ['/n_free', 0]]];
    cmds = cmds.sort { |a, b| a[0] <= b[0] };
    Score(cmds).recordNRT(outputFilePath: file, sampleRate: 44100, headerFormat: "WAV", sampleFormat: "float",
        options: ServerOptions.new.numOutputBusChannels_(2).numInputBusChannels_(0), duration: secs,
        action: { done.test = true; done.signal });
    done.test = false; done.wait;
};
fork {
    ~render.(Ppar([~pk]), 20, ~dir +/+ "kick.wav");
    ~render.(Ppar([~ph]), 20, ~dir +/+ "hoover.wav");
    ~render.(Ppar([~pk, ~ph]), 20, ~dir +/+ "full.wav");
    "ALL DONE".postln; 0.exit;
};
} { |e| e.reportError; "FAILED".postln; 0.exit };
)
