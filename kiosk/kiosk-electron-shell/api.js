const axios = require('axios');

const API_URL = 'http://localhost:3001/api';

async function getDeviceStatus(deviceId) {
  try {
    const response = await axios.get(`${API_URL}/devices/${deviceId}/status`);
    return response.data;
  } catch (error) {
    console.error('Error fetching device status:', error.message);
    return null;
  }
}

async function getDeviceConfig(deviceId) {
    try {
      const response = await axios.get(`${API_URL}/devices/${deviceId}/config`);
      return response.data;
    } catch (error) {
      console.error('Error fetching device config:', error.message);
      return null;
    }
  }

module.exports = {
  getDeviceStatus,
  getDeviceConfig,
};