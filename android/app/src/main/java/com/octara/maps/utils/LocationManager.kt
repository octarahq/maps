package com.octara.maps.utils

import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.os.Looper
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

object LocationManager {
    private val _currentLocation = MutableStateFlow<Pair<Float, Float>?>(null)
    val currentLocation: StateFlow<Pair<Float, Float>?> = _currentLocation.asStateFlow()

    private var isTracking = false
    private var locationCallback: LocationCallback? = null

    fun hasLocationPermission(context: Context): Boolean {
        val hasFine = ContextCompat.checkSelfPermission(
            context, android.Manifest.permission.ACCESS_FINE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED
        val hasCoarse = ContextCompat.checkSelfPermission(
            context, android.Manifest.permission.ACCESS_COARSE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED
        return hasFine || hasCoarse
    }

    @SuppressLint("MissingPermission")
    fun startLocationUpdates(context: Context) {
        if (!hasLocationPermission(context)) return

        val fusedClient = LocationServices.getFusedLocationProviderClient(context.applicationContext)

        fusedClient.lastLocation.addOnSuccessListener { loc ->
            if (loc != null) {
                _currentLocation.value = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
            }
        }

        val cts = CancellationTokenSource()
        fusedClient.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, cts.token)
            .addOnSuccessListener { loc ->
                if (loc != null) {
                    _currentLocation.value = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
                }
            }

        if (isTracking) return

        val request = LocationRequest.Builder(Priority.PRIORITY_BALANCED_POWER_ACCURACY, 10_000L)
            .setMinUpdateIntervalMillis(5_000L)
            .setMinUpdateDistanceMeters(10f)
            .build()

        val callback = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                result.lastLocation?.let { loc ->
                    _currentLocation.value = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
                }
            }
        }
        locationCallback = callback

        try {
            fusedClient.requestLocationUpdates(request, callback, Looper.getMainLooper())
            isTracking = true
        } catch (_: Exception) {
        }
    }

    fun updateLocation(lat: Float, lon: Float) {
        _currentLocation.value = Pair(lat, lon)
    }

    @SuppressLint("MissingPermission")
    suspend fun fetchCurrentLocation(context: Context): Pair<Float, Float>? {
        if (!hasLocationPermission(context)) return null

        val cached = _currentLocation.value
        if (cached != null) return cached

        return suspendCancellableCoroutine { continuation ->
            val fusedClient = LocationServices.getFusedLocationProviderClient(context.applicationContext)
            val cts = CancellationTokenSource()

            fusedClient.getCurrentLocation(
                Priority.PRIORITY_HIGH_ACCURACY,
                cts.token
            ).addOnSuccessListener { loc ->
                if (loc != null) {
                    val coords = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
                    _currentLocation.value = coords
                    continuation.resume(coords)
                } else {
                    continuation.resume(_currentLocation.value)
                }
            }.addOnFailureListener {
                continuation.resume(_currentLocation.value)
            }

            continuation.invokeOnCancellation {
                cts.cancel()
            }
        }
    }
}
