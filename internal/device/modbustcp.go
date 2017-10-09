package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	//	"reflect"
	//	"sync"
	//	log "github.com/Sirupsen/logrus"
	"github.com/yjiong/go_tg120/modbus"
)

type ModbusTcp struct {
	//继承于Device
	Device
	/**************按不同设备自定义*************************/
	Function_code    int
	Starting_address uint16
	Quantity         uint16
	/**************按不同设备自定义*************************/
}

func init() {
	RegDevice["ModbusTcp"] = &ModbusTcp{}
}

func (d *ModbusTcp) NewDev(id string, ele map[string]string) (DeviceRWer, error) {
	ndev := new(ModbusTcp)
	ndev.Device = d.Device.NewDev(id, ele)
	/***********************初始化设备的特有的参数*****************************/
	ndev.Function_code, _ = strconv.Atoi(ele["Function_code"])
	saint, _ := strconv.Atoi(ele["Starting_address"])
	ndev.Starting_address = uint16(saint)
	qint, _ := strconv.Atoi(ele["Quantity"])
	ndev.Quantity = uint16(qint)
	/***********************初始化设备的特有的参数*****************************/
	return ndev, nil
}

func (d *ModbusTcp) GetElement() (dict, error) {
	conn := dict{
		/***********************设备的特有的参数*****************************/
		"devaddr":          d.devaddr,
		"commif":           d.commif,
		"Function_code":    d.Function_code,
		"Starting_address": d.Starting_address,
		"Quantity":         d.Quantity,
		/***********************设备的特有的参数*****************************/
	}
	data := dict{
		"_devid": d.devid,
		"_type":  d.devtype,
		"_conn":  conn,
	}
	return data, nil
}

/***********************设备的参数说明帮助***********************************/
func (d *ModbusTcp) HelpDoc() interface{} {
	conn := dict{
		"devaddr": "设备地址",
		/***********ModbusTcp设备的参数*****************************/
		"commif":           "通信接口,比如 : 192.168.1.20:502",
		"Function_code":    "modbus功能码 : (1,2,3,4,5,6,15,16)",
		"Starting_address": "操作起始地址,uint类型",
		"Quantity":         "寄存器数量,uint类型",
		/***********ModbusTcp设备的参数*****************************/
	}
	r_parameter := dict{
		"_devid": "被读取设备对象的id",
		/***********读取设备的参数*****************************/
		"Function_code":    "modbus功能码 : (1,2,3,4)",
		"Starting_address": "操作起始地址,uint类型",
		"Quantity":         "寄存器数量,uint类型",
		"说明":               "如果没有Function_code,Starting_address,Quantity字段,将按添加该设备时的参数读取设备",
		/***********读取设备的参数*****************************/
	}
	w_parameter := dict{
		"_devid": "被操作设备对象的id",
		/***********操作设备的参数*****************************/
		"Function_code":    "modbus功能码 : (5,6,15,16)",
		"Starting_address": "操作起始地址,uint类型",
		"Quantity":         "寄存器数量,uint类型",
		"value":            "要写入modbus设备的值,功能码为5和6时,值为uint16,功能码为15,16时,值为 [uint8...]",
		/***********操作设备的参数*****************************/
	}
	data := dict{
		"_devid": "添加设备对象的id",
		"_type":  "MudbusTcp", //设备类型
		"_conn":  conn,
	}
	dev_update := dict{
		"request": dict{
			"cmd":  "manager/dev/update.do",
			"data": data,
		},
	}
	readdev := dict{
		"request": dict{
			"cmd":  "do/getvar",
			"data": r_parameter,
		},
	}
	writedev := dict{
		"request": dict{
			"cmd":  "do/setvar",
			"data": w_parameter,
		},
	}
	helpdoc := dict{
		"1.添加设备": dev_update,
		"2.读取设备": readdev,
		"3.操作设备": writedev,
	}
	return helpdoc
}

/***********************设备的参数说明帮助***********************************/

/***************************************添加设备参数检验**********************************************/
func (d *ModbusTcp) CheckKey(ele dict) (bool, error) {

	fc, fc_ok := ele["Function_code"].(json.Number)
	if !fc_ok {
		return false, errors.New(fmt.Sprintf("ModbusTcp device must have int type element 功能码 :Function_code"))
	}
	if fci64, err := fc.Int64(); err != nil || fci64 < 1 || fci64 > 21 {
		return false, errors.New(fmt.Sprintf("Function_code :0 < value < 22 "))
	}
	if _, ok := ele["Starting_address"].(json.Number); !ok {
		return false, errors.New(fmt.Sprintf("ModbusTcp device must have int type element 起始地址 :Starting_address"))
	}
	if _, ok := ele["Quantity"].(json.Number); !ok {
		return false, errors.New(fmt.Sprintf("ModbusTcp device must have int type element 数量 :Quantity"))
	}
	return true, nil
}

/***************************************添加设备参数检验**********************************************/

/***************************************读写接口实现**************************************************/
func (d *ModbusTcp) RWDevValue(rw string, m dict) (ret dict, err error) {
	handler := modbus.NewTCPClientHandler(d.commif)
	slaveid, _ := strconv.Atoi(d.devaddr)
	handler.SlaveId = byte(slaveid)
	handler.Timeout = 1 * time.Second
	ret = map[string]interface{}{}
	ret["_devid"] = d.devid
	err = handler.Connect()
	if err != nil {
		return nil, err
	}
	defer handler.Close()
	function_code := d.Function_code
	start_addr := d.Starting_address
	quantity := d.Quantity
	fc, fc_ok := m["Function_code"].(json.Number)
	sd, sd_ok := m["Starting_address"].(json.Number)
	qt, qt_ok := m["Quantity"].(json.Number)
	if fc_ok && sd_ok && qt_ok {
		fc64, _ := fc.Int64()
		function_code = int(fc64)
		sd64, _ := sd.Int64()
		start_addr = uint16(sd64)
		qt64, _ := qt.Int64()
		quantity = uint16(qt64)
	}
	client := modbus.NewClient(handler)
	var myRfunc func(address, quantity uint16) (results []byte, err error)
	if rw == "r" {
		switch function_code {
		case 1:
			myRfunc = client.ReadCoils
		case 2:
			myRfunc = client.ReadDiscreteInputs
		case 3:
			myRfunc = client.ReadHoldingRegisters
		case 4:
			myRfunc = client.ReadInputRegisters
			//		client.ReadWriteMultipleRegisters
			//		client.ReadFIFOQueue
		default:
			return nil, errors.New(fmt.Sprintf("尚未支持的读操作  Function_code : %d", function_code))
		}
		var results []byte
		results, err = myRfunc(start_addr, quantity)
		if err == nil {
			var retlist []int
			for _, b := range results {
				retlist = append(retlist, int(b))
			}
			ret["Modbus-value"] = retlist
		}
	} else if rw == "w" {
		var results []byte
		var value uint16
		var valuelist []byte
		if v, ok := m["value"].(json.Number); !ok && (function_code == 5 || function_code == 6) {
			return nil, errors.New("write modbus singlecoil or registers need value : uint16")
		} else {
			v64, _ := v.Int64()
			value = uint16(v64)
		}
		if vif, ok := m["value"].([]interface{}); !ok && (function_code == 15 || function_code == 16) {
			return nil, errors.New("write modbus singlecoil or registers need value : [uint8...]")
		} else {
			for _, v := range vif {
				if vi, ok := v.(json.Number); ok {
					vi64, _ := vi.Int64()
					valuelist = append(valuelist, IntToBytes(int(vi64))[3])
				} else {
					return nil, errors.New("write modbus singlecoil or registers need value : [uint8...]")
				}
			}
		}
		switch function_code {
		case 5:
			{
				results, err = client.WriteSingleCoil(start_addr, value)
				if err == nil {
					var retlist []int
					for _, b := range results {
						retlist = append(retlist, int(b))
					}
					ret["Modbus-write"] = retlist
				}
			}
		case 15:
			{
				results, err = client.WriteMultipleCoils(start_addr, quantity, valuelist)
				if err == nil {
					var retlist []int
					for _, b := range results {
						retlist = append(retlist, int(b))
					}
					ret["Modbus-write"] = retlist
				}
			}
		case 6:
			{
				results, err = client.WriteSingleRegister(start_addr, value)
				if err == nil {
					var retlist []int
					for _, b := range results {
						retlist = append(retlist, int(b))
					}
					ret["Modbus-write"] = retlist
				}
			}
		case 16:
			{
				results, err = client.WriteMultipleRegisters(start_addr, quantity, valuelist)
				if err == nil {
					var retlist []int
					for _, b := range results {
						retlist = append(retlist, int(b))
					}
					ret["Modbus-write"] = retlist
				}
			}
		default:
			return nil, errors.New(fmt.Sprintf("尚未支持的写操作  Function_code : %d", function_code))

		}
	}
	return ret, err
}

/***************************************读写接口实现**************************************************/