package com.octara.maps.utils

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

data class IndoorTrackerState(
    val steps: Int = 0,
    val distanceWalkedM: Float = 0f,
    val headingDeg: Float = 0f,
    val isTracking: Boolean = false
)

object IndoorStepTracker : SensorEventListener {
    private val _state = MutableStateFlow(IndoorTrackerState())
    val state: StateFlow<IndoorTrackerState> = _state.asStateFlow()

    private var sensorManager: SensorManager? = null
    private var hasStepDetector = false
    private var stepLengthM = 0.72f

    private var filteredAccel = 9.81f
    private var lastPeakTimeNs = 0L
    private var wasAboveThreshold = false

    private val lastAccel = FloatArray(3)
    private val lastMag = FloatArray(3)
    private var hasAccel = false
    private var hasMag = false
    private var hasRotationVector = false
    private var smoothedHeadingRad: Double? = null

    private val rotationMatrix = FloatArray(9)
    private val orientationAngles = FloatArray(3)

    fun start(context: Context) {
        if (_state.value.isTracking) return
        val sm = context.applicationContext.getSystemService(Context.SENSOR_SERVICE) as? SensorManager
            ?: return
        sensorManager = sm

        val stepDetector = sm.getDefaultSensor(Sensor.TYPE_STEP_DETECTOR)
        hasStepDetector = stepDetector != null
        if (stepDetector != null) {
            sm.registerListener(this, stepDetector, SensorManager.SENSOR_DELAY_UI)
        }

        sm.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)?.let { accel ->
            sm.registerListener(this, accel, SensorManager.SENSOR_DELAY_GAME)
        }

        val rotVec = sm.getDefaultSensor(Sensor.TYPE_ROTATION_VECTOR)
        hasRotationVector = rotVec != null
        if (rotVec != null) {
            sm.registerListener(this, rotVec, SensorManager.SENSOR_DELAY_UI)
        } else {
            sm.getDefaultSensor(Sensor.TYPE_MAGNETIC_FIELD)?.let { mag ->
                sm.registerListener(this, mag, SensorManager.SENSOR_DELAY_UI)
            }
        }

        _state.value = _state.value.copy(isTracking = true)
    }

    fun stop() {
        sensorManager?.unregisterListener(this)
        sensorManager = null
        _state.value = _state.value.copy(isTracking = false)
    }

    fun resetProgress() {
        _state.value = _state.value.copy(steps = 0, distanceWalkedM = 0f)
    }

    fun setStepLength(meters: Float) {
        stepLengthM = meters.coerceIn(0.35f, 1.0f)
    }

    override fun onSensorChanged(event: SensorEvent) {
        when (event.sensor.type) {
            Sensor.TYPE_STEP_DETECTOR -> {
                registerStep()
            }

            Sensor.TYPE_ACCELEROMETER -> {
                System.arraycopy(event.values, 0, lastAccel, 0, 3)
                hasAccel = true

                if (!hasStepDetector) {
                    val mag = sqrt(
                        event.values[0] * event.values[0] +
                            event.values[1] * event.values[1] +
                            event.values[2] * event.values[2]
                    )

                    filteredAccel = filteredAccel * 0.85f + mag * 0.15f
                    val delta = mag - filteredAccel
                    val now = event.timestamp
                    if (delta > 1.35f && !wasAboveThreshold && (now - lastPeakTimeNs) > 320_000_000L) {
                        wasAboveThreshold = true
                        lastPeakTimeNs = now
                        registerStep()
                    } else if (delta < 0.4f) {
                        wasAboveThreshold = false
                    }
                }

                if (!hasRotationVector && hasMag) {
                    if (SensorManager.getRotationMatrix(rotationMatrix, null, lastAccel, lastMag)) {
                        SensorManager.getOrientation(rotationMatrix, orientationAngles)
                        updateSmoothedHeading(orientationAngles[0].toDouble())
                    }
                }
            }

            Sensor.TYPE_MAGNETIC_FIELD -> {
                System.arraycopy(event.values, 0, lastMag, 0, 3)
                hasMag = true
                if (!hasRotationVector && hasAccel) {
                    if (SensorManager.getRotationMatrix(rotationMatrix, null, lastAccel, lastMag)) {
                        SensorManager.getOrientation(rotationMatrix, orientationAngles)
                        updateSmoothedHeading(orientationAngles[0].toDouble())
                    }
                }
            }

            Sensor.TYPE_ROTATION_VECTOR -> {
                SensorManager.getRotationMatrixFromVector(rotationMatrix, event.values)
                SensorManager.getOrientation(rotationMatrix, orientationAngles)
                updateSmoothedHeading(orientationAngles[0].toDouble())
            }
        }
    }

    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}

    private fun registerStep() {
        val cur = _state.value
        _state.value = cur.copy(
            steps = cur.steps + 1,
            distanceWalkedM = cur.distanceWalkedM + stepLengthM
        )
    }

    private fun updateSmoothedHeading(azimuthRad: Double) {
        val prev = smoothedHeadingRad
        val smoothed = if (prev == null) {
            azimuthRad
        } else {
            val s = sin(azimuthRad) * 0.25 + sin(prev) * 0.75
            val c = cos(azimuthRad) * 0.25 + cos(prev) * 0.75
            atan2(s, c)
        }
        smoothedHeadingRad = smoothed
        val deg = ((Math.toDegrees(smoothed) + 360.0) % 360.0).toFloat()
        _state.value = _state.value.copy(headingDeg = deg)
    }
}
