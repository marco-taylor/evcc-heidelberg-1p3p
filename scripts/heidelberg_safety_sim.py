#!/usr/bin/env python3
"""Host fault injection into pinned, unmodified Ethernet firmware functions.

No device access. GPIO, RTU, clock and restart are simulated. Expected failures
are known unsafe policies, not safety approvals. See docs/HEIDELBERG_SAFETY_AUDIT.md.
"""

import argparse
import hashlib
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
import urllib.request

COMMIT = "1e6654ddc7d15faa034e8e2980d6c3aef9f9d238"
SOURCE_SHA256 = "6ecba1351b0fde8af0287aa05d3c2066688df4386e3f472b13f7073a06874fdf"


def extract(source, name):
    """Keep original function text, balancing braces outside comments/literals."""
    masked = re.sub(r'//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'',
                    lambda m: " " * len(m.group()), source, flags=re.S)
    match = re.search(r"^[^\n;{}]*\b" + re.escape(name) + r"\([^;{}]*\)\s*\{", masked, re.M)
    if match is None:
        raise ValueError("function missing: " + name)
    opening = masked.index("{", match.start())
    depth = 1
    end = opening + 1
    while depth:
        depth += (masked[end] == "{") - (masked[end] == "}")
        end += 1
    return source[match.start():end] + "\n"


PRELUDE = r'''
#include <algorithm>
#include <array>
#include <atomic>
#include <cstdint>
#include <iostream>
#include <mutex>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>
#define ESP_LOGI(...) ((void)0)
#define ESP_LOGW(...) ((void)0)
#define ESP_LOGE(...) ((void)0)
#define RTC_NOINIT_ATTR
constexpr int ESP_RST_SW=1, ESP_RST_BROWNOUT=2, ESP_RST_PANIC=3;
int resetReason=ESP_RST_SW;
int esp_reset_reason(){return resetReason;}
// CRC is an environmental stub: cases test the reset-reason gate, not ROM CRC.
uint32_t esp_rom_crc32_le(uint32_t crc,const uint8_t* bytes,size_t length){
  for(size_t i=0;i<length;i++)crc=(crc*33)^bytes[i];
  return crc;
}
constexpr int ESP_OK=0, PIN_1P_IN=36, PIN_3P_IN=39, PIN_1P_OUT=16, PIN_3P_OUT=2;
constexpr int RELAY_OFF=0, RELAY_ON=1, HEC_REG_MAX_CURRENT=261, HEC_REG_REMOTE_LOCK=259;
constexpr int HEC_REG_WATCHDOG=257, HEC_REG_FAILSAFE_CURRENT=262, HEC_MIN_CURRENT=60;
constexpr int HOLDING_REG_OFFSET=256;
uint32_t clockMs=1001;
std::array<int,40> pins{};
std::vector<std::pair<int,int>> outputs;
uint32_t nowMs(){return clockMs;}
int readPin(int pin){return pins.at(pin);}
void writePin(int pin,int value){pins.at(pin)=value; outputs.emplace_back(pin,value);}
void cycleCounterContactorOn(int){}
enum class EventType {SafetyFault, EvccTimeout};
void eventLog(EventType,int,int,int,uint32_t){}
enum class SwitchSource {Evcc, WebPage};
enum class State {Running, SwitchPhases, WaitingForZero, ConfirmedZero, WaitingForOff,
  ConfirmedOff, SwitchedOn, Delay, Fault, Unlocking, Restarting};
struct Restart {};
class PhaseSwitch {
public:
  std::mutex _stateMutex, _cacheMutex, _evccMutex;
  State _state=State::WaitingForOff;
  std::atomic<uint8_t> _desiredPhases{1}, _requestedPhases{0};
  std::atomic<uint16_t> _safetyFaultCode{0};
  std::atomic<const char*> _safetyFaultText{""};
  bool _restartRequested=false, _tcpServerStarted=true, _tcpBridgeWanted=true;
  bool _switchingSupported=true, _firmwareSupported=true, _master_handle=true;
  bool _wallboxSilent=false, _switchOnAnswered=false, _phaseSwitchRequested=false;
  bool _currentWrittenDuringSwitch=false, _restartStopWritten=false, _carQuiet=false;
  bool _evccTimeout=false;
  uint32_t _previous=0, _delay=0, _tcpBridgeLastTryMs=0, _bothFeedbackSinceMs=0;
  uint32_t _invalidFeedbackSinceMs=0, _switchOffDeadlineMs=0, _switchOnDeadlineMs=0;
  uint32_t _wallboxResponseDeadlineMs=0, _zeroLoadDeadlineMs=0, _switchDelay=120000;
  uint32_t _unlockDeadlineMs=0, _lastWallboxOkMs=0, _carQuietUntilMs=0;
  uint32_t _restartDeadlineMs=0, _cachedCurrentWrites=0, _evccTimeouts=0;
  std::atomic<uint32_t> _lastBridgeRequestMs{0};
  std::vector<uint16_t> _holdingRegister=std::vector<uint16_t>(10);
  std::vector<uint16_t> _inputRegister=std::vector<uint16_t>(20);
  uint16_t _reportedStatus=4;
  bool _bootCacheValid=false;
  uint32_t _switchCount=0, _wallboxOutages=0;
  std::array<uint16_t,4> sample{{7,100,100,100}};
  std::array<uint16_t,3> voltages{{230,230,230}};
  std::vector<std::pair<int,int>> writes;
  bool readOK=true, zeroWriteOK=true, lockWriteOK=true, setupOK=true;
  bool silentOnCheck=false;
  int restarts=0;
  int masterWriteHolding(uint16_t reg,uint16_t value){
    writes.emplace_back(reg,value);
    return (reg==261 ? zeroWriteOK : lockWriteOK) ? ESP_OK : -1;
  }
  int masterReadInput(uint16_t reg,uint16_t count,uint16_t* dest){
    if(!readOK) return -1;
    if(reg==10){std::copy_n(voltages.begin(),count,dest);}
    else {std::copy_n(sample.begin(),count,dest);}
    return ESP_OK;
  }
  uint16_t getHoldingRegister(uint16_t reg){return _holdingRegister.at(reg-256);}
  void cacheHolding(uint16_t reg,const uint16_t* v,uint16_t){_holdingRegister.at(reg-256)=*v;}
  void rememberCarState(){}
  void startTcpBridge(){}
  bool validateSetup(){return setupOK;}
  void checkWallboxSilent(){_wallboxSilent=silentOnCheck;}
  void logRefusedConnections(bool){}
  bool updateCachedRegisters(){return false;}
  void logCarState(){}
  void finishSwitchStats(){}
  void abortSwitch(){_state=State::Running;}
  void startSwitch(uint8_t phase,uint8_t){_desiredPhases=phase; _state=State::SwitchPhases;}
  [[noreturn]] void restartNow(){restarts++; throw Restart{};}
  void loop(); void handleRestart(); bool writeRestartStop();
  void enterSafetyFault(uint16_t,const char*); bool hasSafetyFault();
  uint8_t reportedPhases(); bool readActivePhases(uint8_t&); void checkEvccWatchdog();
  void loadBootCache();
};
'''

CASES = r'''
void require(bool condition,const char* message){if(!condition)throw std::runtime_error(message);}
int main(int argc,char**argv){try{
  if(argc!=2)return 2;
  PhaseSwitch p;
  std::string test=argv[1];
  pins.fill(0); pins[PIN_1P_IN]=1;
  p._state=State::Running;
  if(test=="restart_waits_for_zero"){
    p.handleRestart(); require(p.restarts==0 && outputs.empty(),"restart before stop confirmation");
  }else if(test=="restart_confirmed"){
    p.handleRestart(); p.sample={10,0,0,0}; clockMs+=501;
    try{p.handleRestart();}catch(Restart&){}
    require(p.restarts==1,"confirmed stop did not restart");
  }else if(test=="restart_timeout"){
    p.handleRestart(); clockMs+=10001;
    try{p.handleRestart();}catch(Restart&){}
    require(p.restarts==0,"restart occurred despite nonzero current");
  }else if(test=="restart_read_error"){
    p.handleRestart(); p.readOK=false; clockMs+=10001;
    try{p.handleRestart();}catch(Restart&){}
    require(p.restarts==0,"restart occurred without readable current");
  }else if(test=="restart_lock_error"){
    p.lockWriteOK=false;
    require(!p.writeRestartStop(),"failed remote-lock write reported successful");
  }else if(test=="debounce_even"){
    clockMs=1000; uint32_t since=0;
    require(!persistsFor(true,since,100),"100 ms debounce accepted zero elapsed time");
  }else if(test=="debounce_normal"){
    uint32_t since=0;
    require(!persistsFor(true,since,100),"premature debounce");
    clockMs+=100; require(persistsFor(true,since,100),"missing debounce");
  }else if(test=="wrong_feedback"){
    p._desiredPhases=3; p.loop(); clockMs+=101; p.loop();
    require(p.hasSafetyFault(),"opposite single feedback accepted while Running");
  }else if(test=="missing_feedback"){
    pins[PIN_1P_IN]=0; p.loop(); clockMs+=101; p.loop();
    require(p._safetyFaultCode==0xE103 && outputs.size()==2,"missing feedback not latched");
  }else if(test=="both_feedback"){
    pins[PIN_3P_IN]=1; p.loop(); clockMs+=101; p.loop();
    require(p._safetyFaultCode==0xE101 && outputs.size()==2,"both feedback not latched");
  }else if(test=="fault_latched"){
    p.enterSafetyFault(0xE101,"injected"); auto n=outputs.size();
    pins[PIN_1P_IN]=0; p.loop();
    require(p._state==State::Fault && outputs.size()==n,"fault recovered without reset");
  }else if(test=="stop_write_error"){
    p._state=State::SwitchPhases; p.zeroWriteOK=false; p.loop();
    require(outputs.empty() && p._state==State::SwitchPhases,"switch despite failed stop");
  }else if(test=="nonzero_current"){
    p._state=State::WaitingForZero; p._zeroLoadDeadlineMs=clockMs-1; p.loop();
    require(outputs.empty() && p._state==State::WaitingForZero,"switch under nonzero current");
  }else if(test=="zero_read_error"){
    p._state=State::WaitingForZero; p.readOK=false; p.loop();
    require(outputs.empty(),"switch without a current reading");
  }else if(test=="zero_confirmed"){
    p._state=State::WaitingForZero; p.sample={10,0,0,0}; p.loop();
    require(p._state==State::ConfirmedZero && outputs.empty(),"missing stop confirmation");
    p.loop(); require(p._state==State::WaitingForOff && outputs.size()==2,"missing output-off");
  }else if(test=="stuck_contactor"){
    p._state=State::WaitingForOff; p.loop(); clockMs+=8001; p.loop();
    require(p._safetyFaultCode==0xE105,"stuck feedback not latched");
  }else if(test=="off_deadtime"){
    pins[PIN_1P_IN]=0; p._state=State::WaitingForOff; p.loop();
    clockMs+=1999; p.loop(); require(outputs.empty(),"energized before off deadtime");
    clockMs+=1; p.loop(); require(outputs.size()==1,"missing delayed energize");
  }else if(test=="boot_without_request"){
    pins[PIN_1P_IN]=0; p._state=State::WaitingForOff; p.loop(); clockMs+=2000; p.loop();
    require(outputs.empty(),"contactor energized without an external request");
  }else if(test=="boot_current_zero"){
    p._state=State::Delay; p._holdingRegister[5]=160; p.loop();
    require(p.writes.size()>=2 && p.writes[0]==std::make_pair(261,0)
      && p.writes[1]==std::make_pair(259,1),"boot did not restore 0 A before unlock");
  }else if(test=="switch_old_current_capped"){
    p._state=State::Delay; p._phaseSwitchRequested=true; p._holdingRegister[5]=160; p.loop();
    require(p.writes[0]==std::make_pair(261,60),"old current not capped to 6 A");
  }else if(test=="unlock_requires_confirmation"){
    p._state=State::Unlocking; p.readOK=false; p.lockWriteOK=false;
    p._unlockDeadlineMs=clockMs-1; p.loop();
    require(p._state!=State::Running,"Running despite failed unlock/readback");
  }else if(test=="phase_report_is_confirmation"){
    p._state=State::WaitingForZero; p._desiredPhases=3;
    require(p.reportedPhases()==1,"register reports requested 3P while feedback is 1P");
  }else if(test=="voltage_rejects_mixed"){
    uint8_t phases=255; p.voltages={230,230,0};
    require(p.readActivePhases(phases) && phases==0,"mixed voltages reported as valid phases");
  }else if(test=="watchdog_zero"){
    p._holdingRegister[1]=100; p._holdingRegister[6]=0;
    p._lastBridgeRequestMs=1; p.checkEvccWatchdog();
    require(p.writes.size()==1 && p.writes[0]==std::make_pair(261,0),"watchdog did not apply 0 A");
  }else if(test=="software_cache" || test=="brownout_cache" || test=="panic_cache"){
    s_bootCache={}; s_bootCache.magic=BOOT_CACHE_MAGIC; s_bootCache.desiredPhases=3;
    s_bootCache.crc=bootCacheCrc(s_bootCache);
    resetReason=test=="software_cache" ? ESP_RST_SW : (test=="brownout_cache" ? ESP_RST_BROWNOUT : ESP_RST_PANIC);
    p.loadBootCache();
    require(p._bootCacheValid==(resetReason==ESP_RST_SW),"reset reason gate changed");
    require(p._desiredPhases==(resetReason==ESP_RST_SW ? 3 : 1),"reset phase selection changed");
  }else if(test=="watchdog_nonzero"){
    p._holdingRegister[1]=100; p._holdingRegister[6]=160;
    p._lastBridgeRequestMs=1; p.checkEvccWatchdog();
    require(p.writes.size()==1 && p.writes[0]==std::make_pair(261,160),"failsafe policy changed");
  }else{return 2;}
  return 0;
  }catch(const std::exception& error){std::cerr<<error.what()<<"\n"; return 1;}
}
'''


class SafetySimulation(unittest.TestCase):
    binary = None

    def setUp(self):
        name = self._testMethodName.removeprefix("test_")
        self.result = subprocess.run([self.binary, name], capture_output=True, text=True, timeout=5)
        # setUp errors are never swallowed by expectedFailure.
        self.assertIn(self.result.returncode, (0, 1), self.result.stderr)
        if name in KNOWN_MESSAGES and self.result.returncode == 1:
            self.assertEqual(self.result.stderr.strip(), KNOWN_MESSAGES[name])

    def run_case(self, name):
        self.assertEqual(self.result.returncode, 0, f"{name}: {self.result.stderr}")


REGRESSION = ["restart_waits_for_zero", "restart_confirmed", "debounce_normal",
        "missing_feedback", "both_feedback", "fault_latched", "stop_write_error",
        "nonzero_current", "zero_read_error", "zero_confirmed", "stuck_contactor",
        "off_deadtime", "boot_current_zero", "switch_old_current_capped",
        "voltage_rejects_mixed", "watchdog_zero", "watchdog_nonzero",
        "software_cache", "brownout_cache", "panic_cache"]
KNOWN_GAPS = ["restart_timeout", "restart_read_error", "restart_lock_error",
              "debounce_even", "wrong_feedback", "boot_without_request",
              "unlock_requires_confirmation", "phase_report_is_confirmation"]
KNOWN_MESSAGES = {
    "restart_timeout": "restart occurred despite nonzero current",
    "restart_read_error": "restart occurred without readable current",
    "restart_lock_error": "failed remote-lock write reported successful",
    "debounce_even": "100 ms debounce accepted zero elapsed time",
    "wrong_feedback": "opposite single feedback accepted while Running",
    "boot_without_request": "contactor energized without an external request",
    "unlock_requires_confirmation": "Running despite failed unlock/readback",
    "phase_report_is_confirmation": "register reports requested 3P while feedback is 1P",
}


def test_case(name):
    def test(self):
        self.run_case(name)
    test.__name__ = "test_" + name
    return test


for case in REGRESSION + KNOWN_GAPS:
    method = test_case(case)
    if case in KNOWN_GAPS:
        method = unittest.expectedFailure(method)
    setattr(SafetySimulation, "test_" + case, method)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--firmware-root", type=Path)
    source.add_argument("--fetch-pinned", action="store_true")
    args = parser.parse_args()
    if args.firmware_root:
        raw = (args.firmware_root / "src/switch.cpp").read_bytes()
    else:
        url = f"https://raw.githubusercontent.com/jetronic18s/heidelberg-1p3p/{COMMIT}/src/switch.cpp"
        with urllib.request.urlopen(url, timeout=30) as response:
            raw = response.read()
    if hashlib.sha256(raw).hexdigest() != SOURCE_SHA256:
        raise ValueError("firmware source hash mismatch; review the new source before updating the pin")
    source = raw.decode()
    print(f"Firmware {COMMIT}, SHA256 {SOURCE_SHA256}", flush=True)
    constants = source[source.index("static constexpr uint32_t FEEDBACK_DEBOUNCE_MS"):
                       source.index("// Register cache kept across")]
    boot_cache = source[source.index("static constexpr uint32_t BOOT_CACHE_MAGIC"):
                        source.index("struct TcpClientContext")]
    functions = ["msSince", "pastDeadline", "PhaseSwitch::loop",
                 "PhaseSwitch::handleRestart", "PhaseSwitch::writeRestartStop",
                 "PhaseSwitch::enterSafetyFault", "PhaseSwitch::hasSafetyFault",
                 "PhaseSwitch::reportedPhases", "PhaseSwitch::readActivePhases",
                 "PhaseSwitch::checkEvccWatchdog", "PhaseSwitch::loadBootCache"]
    # The fixture defaults used in the boot tests are checked against the constructor.
    for initializer in ("_state(State::WaitingForOff)", "_desiredPhases(1)", "_phaseSwitchRequested(false)"):
        if initializer not in source:
            raise ValueError("constructor default changed: " + initializer)
    with tempfile.TemporaryDirectory(prefix="heidelberg-safety-") as folder:
        path = Path(folder)
        cpp = path / "simulation.cpp"
        cpp.write_text(PRELUDE + constants + boot_cache + "\n".join(extract(source, f) for f in functions) + CASES)
        SafetySimulation.binary = str(path / "simulation")
        subprocess.run(["g++", "-std=c++17", "-Wall", "-Wextra", "-Werror", "-Wno-unused-parameter",
                        str(cpp), "-o", SafetySimulation.binary], check=True)
        suite = unittest.defaultTestLoader.loadTestsFromTestCase(SafetySimulation)
        result = unittest.TextTestRunner(verbosity=2).run(suite)
        print(f"Known safety gaps: {len(result.expectedFailures)}. This is not a safety approval.")
        return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
    raise SystemExit(main())
